/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package pod

import (
	"bufio"
	"container/list"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
	log "github.com/sirupsen/logrus"

	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/types"
	utils "github.com/paypal/dce-go/utils/wait"
	"github.com/paypal/gorealis/gen-go/apache/aurora"
)

const (
	OUTPUT_DELIMITER        = ","
	PORT_SEPARATOR          = ":"
	PRIM_INSPECT_RESULT_LEN = 7
	INSPECT_RESULT_LEN      = 6
)

var ComposeExcutorDriver executor.ExecutorDriver
var PodStatus = &types.PodStatus{
	Status: types.POD_STAGING,
}
var ComposeFiles []string
var ComposeTaskInfo *mesos.TaskInfo

var HealthCheckListId = make(map[string]bool)
var PodContainers []string
var SinglePort bool

// Check exit code of all the containers in the pod.
// If all the exit codes are zero, then assign zero as pod's exit code,
// otherwise assign the first non zero exit code to pod
func CheckPodExitCode(files []string) (int, error) {
	containerIds, err := GetPodContainerIds(files)
	if err != nil {
		log.Errorln("Error retrieving container ids")
		return 1, err
	}

	for _, containerId := range containerIds {
		exitCode, err := checkContainerExitCode(containerId)

		if err != nil {
			return 1, err
		}

		if exitCode != 0 {
			return exitCode, nil
		}
	}
	return 0, nil
}

// Check exit code of container
func checkContainerExitCode(containerId string) (int, error) {
	out, err := exec.Command("docker", "inspect",
		"--format='{{.State.ExitCode}}'", containerId).Output()
	_out := strings.Trim(strings.Trim(string(out[:]), "\n"), "'")
	log.Printf("Check Pod Exit Code : Container %s ExitCode : %v\n", containerId, _out)
	if err != nil {
		log.Errorf("Error retrieving container exit code of container : %s, %s\n", containerId, err.Error())
		return 1, err
	}

	exitCode, err := strconv.Atoi(_out)
	if err != nil {
		log.Fatalf("Error converting string to int : %s\n", err.Error())
	}
	return exitCode, nil
}

// Generate cmd parts
// docker-compose parts
// example : docker-compose -f compose.yaml up
// "docker-compose" will be the main cmd, "-f compose.yaml up" will be parts and return as an array
func GenerateCmdParts(files []string, cmd string) ([]string, error) {
	if config.EnableVerbose() {
		cmd = " --verbose" + cmd
	}
	var s string
	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			return nil, err
		} else {
			s += "-f " + file + " "
		}
	}
	s = strings.TrimSpace(s) + cmd
	return strings.Fields(s), nil
}

// Get set of containers id in pod
// docker-compose -f docker-compose.yaml ps -q
func GetPodContainerIds(files []string) ([]string, error) {
	var containerIds []string

	parts, err := GenerateCmdParts(files, " ps -q")
	if err != nil {
		log.Errorln("Error generating compose cmd parts")
		return nil, err
	}

	out, err := utils.RetryCmd(config.GetMaxRetry(), exec.Command("docker-compose", parts...))
	if err != nil {
		log.Errorf("GetContainerIds : Error executing cmd docker-compose ps %#v", err)
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out[:])))
	for scanner.Scan() {
		containerIds = append(containerIds, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Errorln(os.Stderr, "reading standard input:", err)
	}
	return containerIds, nil
}

// GetContainerIdsByServices get container ids by a list of services
func GetContainerIdsByServices(files, services []string) ([]string, error) {
	var ids []string
	for _, s := range services {
		id, err := GetContainerIdByService(files, s)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// GetContainerIdByService does query container id by service name
func GetContainerIdByService(files []string, service string) (string, error) {
	logger := log.WithFields(log.Fields{
		"service": service,
		"func":    "GetContainerIdByService",
	})

	// Return err if service name is empty
	if service == "" {
		err := fmt.Errorf("service name can't be empty")
		return "", err
	}

	// Generate cmd -- docker-compose -f [file] ps -q [service]
	parts, err := GenerateCmdParts(files, " ps -q "+service)
	if err != nil {
		logger.Errorf("POD_GENERATE_COMPOSE_PARTS_FAIL -- %v", err)
		return "", err
	}

	// Run cmd to get container id
	cmd := exec.Command("docker-compose", parts...)
	logger.Printf("Command to get container id by service name: %s", cmd.Args)

	out, err := utils.RetryCmd(config.GetMaxRetry(), cmd)
	if err != nil {
		logger.Errorf("Error getting container id by service: %v", err)
		return "", err
	}

	// Scan output
	var id string
	scanner := bufio.NewScanner(strings.NewReader(string(out[:])))
	for scanner.Scan() {
		id += scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		logger.Errorln(os.Stderr, "stderr: ", err)
		return "", err
	}

	logger.Printf("container id: %s", id)
	return id, nil
}

// docker-compose -f docker-compose.yaml ps
func GetPodDetail(files []string, primaryContainerId string, healthcheck bool) {
	parts, err := GenerateCmdParts(files, " ps")
	if err != nil {
		log.Errorln("Error generating cmd parts")
	}

	//out, err := exec.Command("docker-compose", parts...).Output()
	out, err := utils.RetryCmd(config.GetMaxRetry(), exec.Command("docker-compose", parts...))
	if err != nil {
		log.Errorf("GetPodDetail : Error executing cmd docker-compose ps %#v", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out[:])))
	for scanner.Scan() {
		log.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Errorln(os.Stderr, "reading standard input:", err)
	}

	if primaryContainerId != "" {
		primaryContainerDetail, err := InspectContainerDetails(primaryContainerId, healthcheck)
		if err != nil {
			log.Errorln("Error retrieving primary container detail : ", err.Error())
		}
		log.Printf("Inspect primary container : %s , Name: %s, health status: %s, exit code : %v, is running : %v",
			primaryContainerId, primaryContainerDetail.Name, primaryContainerDetail.HealthStatus,
			primaryContainerDetail.ExitCode, primaryContainerDetail.IsRunning)
	}
}

// Get port range assigned by mesos
func GetPorts(taskInfo *mesos.TaskInfo) *list.Element {
	var ports list.List
	resources := taskInfo.GetResources()
	for _, resource := range resources {
		if resource.GetName() == types.PORTS {
			ranges := resource.GetRanges()
			for _, r := range ranges.GetRange() {
				begin := r.GetBegin()
				end := r.GetEnd()
				log.Println("Port Range : begin: ", begin)
				log.Println("Port range : end: ", end)
				for i := begin; i < end+1; i++ {
					ports.PushBack(i)
				}
			}
		}
	}
	return ports.Front()
}

// Launch pod
// docker-compose up
func LaunchPod(files []string) string {
	log.Println("====================Launch Pod====================")

	parts, err := GenerateCmdParts(files, " up -d")
	if err != nil {
		log.Printf("POD_GENERATE_COMPOSE_PARTS_FAIL -- %v", err)
		return types.POD_FAILED
	}

	cmd := exec.Command("docker-compose", parts...)
	log.Printf("Launch Pod : Command to launch task : %v", cmd.Args)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%d", types.COMPOSE_HTTP_TIMEOUT, config.GetComposeHttpTimeout()))

	go dockerLogToStdout(files)

	err = cmd.Run()
	if err != nil {
		log.Printf("POD_LAUNCH_FAIL -- Error running launch task command : %v", err)
		return types.POD_FAILED
	}

	return types.POD_STARTING
}

func dockerLogToStdout(files []string) {
	parts, err := GenerateCmdParts(files, " logs --follow --no-color")
	if err != nil {
		log.Printf("POD_GENERATE_COMPOSE_PARTS_FAIL -- %v", err)
	}

	cmd := exec.Command("docker-compose", parts...)
	log.Printf("Command to print container log: %v", cmd.Args)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_, err = utils.RetryCmdLogs(cmd)
	if err != nil {
		log.Printf("POD_LAUNCH_LOG_FAIL -- Error running cmd %s\n", cmd.Args)
	}
}

// Stop pod
// docker-compose stop
func StopPod(files []string) error {
	log.Println("====================Stop Pod====================")

	//get stop timeout from config
	timeout := config.GetStopTimeout()
	parts, err := GenerateCmdParts(files, " stop -t "+timeout)
	if err != nil {
		log.Printf("POD_GENERATE_COMPOSE_PARTS_FAIL -- %v", err)
		return err
	}

	cmd := exec.Command("docker-compose", parts...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Stop Pod : Command to stop task : %s", cmd.Args)

	err = cmd.Run()
	if err != nil {
		log.Printf("POD_STOP_FAIL --", err.Error())
		err = ForceKill(files)
		if err != nil {
			log.Printf("POD_STOP_FORCE_FAIL -- Error in force pod kill : %v", err)
			return err
		}
	}

	if network, ok := config.GetNetwork(); ok {
		if network.PreExist {
			return nil
		}
	}

	// skip removing network if network mode is host
	// RM_INFRA_CONTAINER is set as true if network mode is true during yml parsing
	if config.GetConfig().GetBool(types.RM_INFRA_CONTAINER) {
		return nil
	}

	// Get infra container id
	infraContainerId, err := GetContainerIdByService(ComposeFiles, types.INFRA_CONTAINER)
	if err != nil {
		log.Errorf("Error getting container id of service %s: %v", types.INFRA_CONTAINER, err)
		return nil
	}

	networkName, err := GetContainerNetwork(infraContainerId)
	if err != nil {
		log.Errorf("Failed to clean up network :%v", err)
	}
	err = RemoveNetwork(networkName)
	if err != nil {
		log.Printf("POD_CLEAN_NETWORK_FAIL -- %v", err)
	}
	return nil
}

// remove pod volume
// docker-compose down -v
func RemovePodVolume(files []string) error {
	log.Println("====================Remove Pod Volume====================")
	parts, err := GenerateCmdParts(files, " down -v")
	if err != nil {
		log.Printf("POD_GENERATE_COMPOSE_PARTS_FAIL -- %v", err)
		return err
	}

	cmd := exec.Command("docker-compose", parts...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Println("Remove Pod Volume: Command to rm volume : docker-compose ", parts)

	err = cmd.Run()
	if err != nil {
		log.Printf("POD_CLEAN_VOLUME_FAIL -- %v", err)
		return err
	}

	return nil
}

// remove pod images
// docker-compose down --rmi
func RemovePodImage(files []string) error {
	log.Println("====================Remove Pod Images====================")
	parts, err := GenerateCmdParts(files, " down --rmi all")
	if err != nil {
		log.Printf("POD_GENERATE_COMPOSE_PARTS_FAIL -- %v", err)
		return err
	}

	cmd := exec.Command("docker-compose", parts...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Println("Remove Pod Image: Command to rm images : docker-compose ", parts)

	err = cmd.Run()
	if err != nil {
		log.Errorln("POD_CLEAN_IMAGE_FAIL -- probably images are in used by other containers")
	}

	return nil
}

func GetContainerNetwork(id string) (string, error) {
	//cmd := exec.Command("docker", "inspect", "--format='{{.HostConfig.NetworkMode}}'", name)
	//out, err := utils.RetryCmd(config.GetMaxRetry(), cmd)

	out, err := exec.Command("docker", "inspect", "--format='{{.HostConfig.NetworkMode}}'", id).Output()
	if err != nil {
		log.Errorf("Error retrieving container network mode : %s , %s\n", id, err.Error())
		return "", err
	}

	network := strings.Replace(strings.TrimSuffix(string(out[:]), "\n"), "'", "", -1)
	log.Printf("Get container %s network : %s\n", id, network)
	return network, err
}

func RemoveNetwork(name string) error {
	log.Println("====================Remove network====================")
	cmd := exec.Command("docker", "network", "rm", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Errorf("Error in rm network : %s , %s\n", name, err.Error())
	}
	return err
}

// Force kill pod
// docker kill -f
func ForceKill(files []string) error {
	log.Println("====================Force Kill Pod====================")
	parts, err := GenerateCmdParts(files, " kill")
	if err != nil {
		log.Printf("POD_GENERATE_COMPOSE_PARTS_FAIL -- %v", err)
		return err
	}

	cmd := exec.Command("docker-compose", parts...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Println("Kill Pod : Command to kill task : docker-compose ", parts)

	err = cmd.Run()
	if err != nil {
		log.Printf("POD_FORCE_KILL_FAIL -- %v", err)
		return err
	}
	return nil
}

// pull image
// docker-compose pull
func PullImage(files []string) error {
	log.Println("====================Pull Image====================")

	parts, err := GenerateCmdParts(files, " pull")
	if err != nil {
		log.Printf("POD_GENERATE_COMPOSE_PARTS_FAIL -- %v", err)
		return err
	}

	cmd := exec.Command("docker-compose", parts...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Println("Pull Image : Command to pull images : docker-compose ", parts)

	err = cmd.Start()
	if err != nil {
		log.Printf("POD_PULL_IMAGE_FAIL	-- %v ", err)
		return err
	}

	err = utils.WaitCmd(config.GetLaunchTimeout()*time.Millisecond, &types.CmdResult{
		Command: cmd,
	})
	if err != nil {
		return err
	}
	return nil
}

//CheckContainer does check container details
//return healthy,run,err
func CheckContainer(containerId string, healthCheck bool) (string, bool, int, error) {
	containerDetail, err := InspectContainerDetails(containerId, healthCheck)
	if err != nil {
		log.Printf("CheckContainer : Error inspecting container with id : %s, %v", containerId, err.Error())
		return types.UNHEALTHY, containerDetail.IsRunning, containerDetail.ExitCode, err
	}

	if containerDetail.ExitCode != 0 {
		log.Printf("CheckContainer : Container %s is finished with exit code %v\n", containerId, containerDetail.ExitCode)
		return types.UNHEALTHY, containerDetail.IsRunning, containerDetail.ExitCode, nil
	}

	if healthCheck {
		if containerDetail.IsRunning {
			//log.Printf("CheckContainer : Primary container %s is running , %s\n", containerId, containerDetail.HealthStatus)
			return containerDetail.HealthStatus, containerDetail.IsRunning, containerDetail.ExitCode, nil
		}
		return containerDetail.HealthStatus, containerDetail.IsRunning, containerDetail.ExitCode, nil
	}

	if containerDetail.IsRunning {
		//log.Printf("CheckContainer : Regular container %s is running\n", containerId)
		return types.HEALTHY, containerDetail.IsRunning, containerDetail.ExitCode, nil
	}

	return types.HEALTHY, containerDetail.IsRunning, containerDetail.ExitCode, nil
}

func GetDockerPorts(containerId string, privatePort string) (string, error) {
	out, err := utils.RetryCmd(config.GetMaxRetry(), exec.Command("docker", "port", containerId, privatePort))
	if err != nil {
		log.Printf("Error inspecting container dynamic ports : %v", err)
		return "", err
	}
	log.Printf("Get Container Dynamic Port : %s", string(out))
	if ports := strings.Split(string(out), PORT_SEPARATOR); len(ports) > 1 {
		return strings.TrimSuffix(ports[1], "\n"), nil
	}
	return "", err
}

// docker inspect
func InspectContainerDetails(containerId string, healthcheck bool) (types.ContainerStatusDetails, error) {
	var containerStatusDetails types.ContainerStatusDetails
	var out []byte
	var err error
	if healthcheck {
		out, err = utils.RetryCmd(config.GetMaxRetry(), exec.Command("docker", "inspect",
			"--format='{{.State.Pid}},{{.State.Running}},{{.State.ExitCode}},{{.State.Health.Status}},{{.RestartCount}},{{.HostConfig.RestartPolicy.MaximumRetryCount}},{{.Name}}'",
			containerId))
	} else {
		out, err = utils.RetryCmd(config.GetMaxRetry(), exec.Command("docker", "inspect",
			"--format='{{.State.Pid}},{{.State.Running}},{{.State.ExitCode}},{{.RestartCount}},{{.HostConfig.RestartPolicy.MaximumRetryCount}},{{.Name}}'",
			containerId))
	}
	if err != nil {
		log.Printf("Error inspecting container details : %v \n", err)
		return containerStatusDetails, err
	}

	containerStatusDetails, err = ParseToContainerDetail(string(out[:]), healthcheck)
	if err != nil {
		log.Printf("Error parsing output to container details : %v \n", err)
		return containerStatusDetails, err
	}

	containerStatusDetails.SetContainerId(containerId)

	log.Debugf("Inspect container : %s , Name: %s, health status: %s, exit code : %v, is running : %v\n",
		containerId, containerStatusDetails.Name, containerStatusDetails.HealthStatus,
		containerStatusDetails.ExitCode, containerStatusDetails.IsRunning)

	if containerStatusDetails.HealthStatus == types.UNHEALTHY || containerStatusDetails.ExitCode != 0 {
		log.Printf("Inspect container : %s , Name: %s, health status: %s, exit code : %v, is running : %v\n",
			containerId, containerStatusDetails.Name, containerStatusDetails.HealthStatus,
			containerStatusDetails.ExitCode, containerStatusDetails.IsRunning)
	}

	return containerStatusDetails, nil
}

func ParseToContainerDetail(output string, healthcheck bool) (types.ContainerStatusDetails, error) {
	var containerStatusDetails types.ContainerStatusDetails
	array := strings.Split(output, OUTPUT_DELIMITER)
	if healthcheck {
		if len(array) != PRIM_INSPECT_RESULT_LEN {
			err := errors.New("Mismatch with expected inspect result")
			return containerStatusDetails, err
		}
		pid, _ := strconv.Atoi(array[0])
		running, _ := strconv.ParseBool(array[1])
		exitCode, _ := strconv.Atoi(array[2])
		restartCount, _ := strconv.Atoi(array[4])
		maxRetryCount, _ := strconv.Atoi(array[5])
		containerStatusDetails = types.ContainerStatusDetails{
			ComposeTaskId: nil,
			Pid:           pid,
			ContainerId:   "",
			IsRunning:     running,
			ExitCode:      exitCode,
			HealthStatus:  array[3],
			RestartCount:  restartCount,
			MaxRetryCount: maxRetryCount,
			Name:          array[6],
		}
		return containerStatusDetails, nil
	}
	if len(array) != INSPECT_RESULT_LEN {
		err := errors.New("Mismatch with expected inspect result")
		return containerStatusDetails, err
	}
	pid, _ := strconv.Atoi(array[0])
	running, _ := strconv.ParseBool(array[1])
	exitCode, _ := strconv.Atoi(array[2])
	restartCount, _ := strconv.Atoi(array[3])
	maxRetryCount, _ := strconv.Atoi(array[4])
	containerStatusDetails = types.ContainerStatusDetails{
		ComposeTaskId: nil,
		Pid:           pid,
		ContainerId:   "",
		IsRunning:     running,
		ExitCode:      exitCode,
		HealthStatus:  "",
		RestartCount:  restartCount,
		MaxRetryCount: maxRetryCount,
		Name:          array[5],
	}

	return containerStatusDetails, nil
}

// get label by name
func GetLabel(key string, taskInfo *mesos.TaskInfo) string {
	var labelsList []*mesos.Label
	labelsList = taskInfo.GetLabels().GetLabels()
	var label *mesos.Label
	for _, label = range labelsList {
		if label.GetKey() == key {
			return label.GetValue()
		}
		if strings.Contains(label.GetKey(), key) && strings.Contains(label.GetKey(), ".") {
			arr := strings.Split(label.GetKey(), ".")
			if arr[len(arr)-1] == key {
				return label.GetValue()
			}
		}
	}
	return ""
}

// Read pod status
func GetPodStatus() string {
	PodStatus.RLock()
	defer PodStatus.RUnlock()
	return PodStatus.Status
}

// Set pod status
func SetPodStatus(status string) {
	PodStatus.Lock()
	PodStatus.Status = status
	PodStatus.Unlock()
	log.Printf("Update Status : Update podStatus as %s", status)
}

func SendPodStatus(status string) {
	logger := log.WithFields(log.Fields{
		"status": status,
		"func":   "pod.SendPodStatus",
	})
	curntPodStatus := GetPodStatus()
	if curntPodStatus == types.POD_FAILED || curntPodStatus == types.POD_KILLED ||
		curntPodStatus == types.POD_FINISHED || curntPodStatus == status {
		logger.Printf("Task has already been killed or failed or finished or updated as required status: %s", curntPodStatus)
	}

	SetPodStatus(status)

	switch status {
	case types.POD_RUNNING:
		SendMesosStatus(ComposeExcutorDriver, ComposeTaskInfo.GetTaskId(), mesos.TaskState_TASK_RUNNING.Enum())
	case types.POD_FINISHED:
		// Stop pod to kill system proxy container
		if len(PodContainers) > 0 {
			logger.Printf("Stop containers still running in the pod: %v", PodContainers)
			err := StopPod(ComposeFiles)
			if err != nil {
				logger.Errorf("Error stop pod: %v", err)
			}
		}
		SendMesosStatus(ComposeExcutorDriver, ComposeTaskInfo.GetTaskId(), mesos.TaskState_TASK_FINISHED.Enum())
	case types.POD_FAILED:
		err := StopPod(ComposeFiles)
		if err != nil {
			logger.Errorf("Error cleaning up pod : %v\n", err.Error())
		}
		SendMesosStatus(ComposeExcutorDriver, ComposeTaskInfo.GetTaskId(), mesos.TaskState_TASK_FAILED.Enum())
	case types.POD_PULL_FAILED:
		SendMesosStatus(ComposeExcutorDriver, ComposeTaskInfo.GetTaskId(), mesos.TaskState_TASK_FAILED.Enum())
	}
}

//Update mesos and pod status
func SendMesosStatus(driver executor.ExecutorDriver, taskId *mesos.TaskID, state *mesos.TaskState) error {
	runStatus := &mesos.TaskStatus{
		TaskId: taskId,
		State:  state,
	}
	_, err := driver.SendStatusUpdate(runStatus)
	if err != nil {
		log.Errorf("Error updating mesos task status : %v", err.Error())
		return err
	}

	log.Printf("Update Status : Update Mesos task state as %s", state.String())

	time.Sleep(200 * time.Millisecond)
	if state.Enum().String() == mesos.TaskState_TASK_FAILED.Enum().String() ||
		state.Enum().String() == mesos.TaskState_TASK_FINISHED.Enum().String() {
		log.Println("====================Stop ExecutorDriver====================")
		driver.Stop()
	}
	return nil
}

// Wait for pod running/finished until timeout or failed
func WaitOnPod(ctx *context.Context) {
	select {
	case <-(*ctx).Done():
		if (*ctx).Err() == context.DeadlineExceeded {
			log.Println("POD_LAUNCH_TIMEOUT")
			if dump, ok := config.GetConfig().GetStringMap("dockerdump")["enable"].(bool); ok && dump {
				DockerDump()
			}
			SendPodStatus(types.POD_FAILED)
		} else if (*ctx).Err() == context.Canceled {
			log.Println("Stop wait on pod, since pod is running/finished/failed")
		}
	}
}

// DockerDump does dump docker.pid and docker-containerd.pid and docker log if docker dump is enabled in config
func DockerDump() {
	log.Println(" ######## Begin dockerDump ######## ")

	dockerDumpPath := config.GetConfigSection("dockerdump")["dumppath"]
	if dockerDumpPath == "" {
		log.Println("docker dump path can't find in config, set it as /home/ubuntu")
		dockerDumpPath = "/home/ubuntu"
	}

	dockerdPath := config.GetConfigSection("dockerdump")["dockerpidfile"]
	if dockerdPath == "" {
		log.Println("dockerd pid path can't find in config, set it as /var/run/docker.pid")
		dockerdPath = "/var/run/docker.pid"
	}

	containerdPath := config.GetConfigSection("dockerdump")["containerpidfile"]
	if containerdPath == "" {
		log.Println("containerd pid path can't find in config, set it as /run/docker/libcontainerd/docker-containerd.pid")
		containerdPath = "/run/docker/libcontainerd/docker-containerd.pid"
	}

	dockerlogPath := config.GetConfigSection("dockerdump")["dockerlogpath"]
	if dockerlogPath == "" {
		log.Println("docker log path can't find in config, set it as /var/log/upstart/docker.log")
		dockerlogPath = "/var/log/upstart/docker.log"
	}

	//STEP1 -- kill docker pid
	f, err := ioutil.ReadFile(dockerdPath)
	if err != nil {
		log.Errorf("Error opening file %s : %v", dockerDumpPath, err)
		return
	} else {
		pid := string(f)
		log.Println(" docker pid: ", pid)

		cmdDocker := exec.Command("kill", "-USR1", pid)
		log.Printf("Cmd to kill docker pid: %s", cmdDocker.Path)
		cmdDocker.Stdout = os.Stdout
		cmdDocker.Stderr = os.Stderr
		err = cmdDocker.Run()
		if err != nil {
			log.Errorf("Error running cmd to kill -USR1 dockerPid: %v", err)
			return
		} else {
			log.Println(" docker KILL -USR1 dockerPid completed... ")
		}
	}

	//STEP2 --  kill containerd pid
	f, err = ioutil.ReadFile(containerdPath)
	if err != nil {
		log.Errorf("Error opening file %s : %v\n", containerdPath, err)
		return
	} else {
		dockerContainerdPid := string(f)
		log.Println(" DockerContainerdPid: ", dockerContainerdPid)
		cmdContainerd := exec.Command("kill", "-USR1", dockerContainerdPid)
		log.Printf("Cmd to kill containerd pid: %s", cmdContainerd.Path)
		cmdContainerd.Stdout = os.Stdout
		cmdContainerd.Stderr = os.Stderr
		err = cmdContainerd.Run()
		if err != nil {
			log.Errorf("Error running cmd to kill containerd pid: %v", err)
			return
		} else {
			log.Println(" docker KILL -USR1 dockerContainerdPid completed... ")
		}

	}

	// sleep for 15 seconds
	log.Println(fmt.Sprintf("Sleep 15 seconds from: %s", time.Now().Format("2006-01-02 15:04:05")))
	time.Sleep(15 * time.Second)

	timeNow := time.Now()
	fmtTime := timeNow.Format("2006-01-02 15:04:05")

	//STEP3 --  copy docker log
	cmdCpDockerLog := exec.Command("cp", dockerlogPath, dockerDumpPath+"/docker.log."+fmtTime)
	log.Printf("Cmd to copy docker log: %s", cmdCpDockerLog.Path)
	err = cmdCpDockerLog.Run()
	if err != nil {
		log.Errorf("Error running cmd to copy docker log: %v", err)
	} else {
		log.Println(" CP docker.log complete ...")
	}

	//STEP4 -- copy log of docker info
	outDockerInfo, err := exec.Command("docker", "info").Output()
	log.Println("Cmd : docker info")
	if err != nil {
		log.Errorf("Error running cmd to get docker info: %v", err)
	} else {
		f, err := os.OpenFile(dockerDumpPath+"/docker.info."+fmtTime, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Errorln("Error opening file")
		}
		defer f.Close()
		_, err = f.Write(outDockerInfo)
		if err != nil {
			log.Errorf(" Failed file write: %s/docker.info.. error:%v", dockerDumpPath, err)
		}
		log.Println("create docker.info log complete....")
	}

	//STEP5 --  copy log of vmstat
	outVmstat, err := exec.Command("vmstat").Output()
	log.Println("Cmd : vmstat")
	if err != nil {
		log.Errorf("Error running cmd vmstat: %v", err)
	} else {
		f, err := os.OpenFile(dockerDumpPath+"/vmstat.info."+fmtTime, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Errorln("Error opening file")
		}
		defer f.Close()
		_, err = f.Write(outVmstat)
		if err != nil {
			log.Errorf(" Failed file write: %s/vmstat.info.. error:%v", dockerDumpPath, err)
		}
	}

	//STEP6 --  restart docker
	/*restart := exec.Command("restart", "docker")
	log.Println("Cmd : restart docker")
	err = restart.Run()
	if err != nil {
		log.Errorf("Error running cmd to restart docker: %v", err)
	}*/

	log.Println(" ######## End dockerDump ######## ")
}

// healthCheck includes health checking for primary container and exit code checking for other containers
func HealthCheck(files []string, podServices map[string]bool, out chan<- string) {
	logger := log.WithFields(log.Fields{
		"func": "HealthCheck",
	})
	logger.Println("====================Health Check====================", len(podServices))
	logger.Printf("pod service list: %v", podServices)
	var err error
	var containers []string
	var healthCount int

	interval := time.Duration(config.GetPollInterval())

	// Convert pod services from map to array
	var services []string
	for name := range podServices {
		services = append(services, name)
	}

	logger.Printf("service list: %v", services)

	// Start checking containers are running and healthy ONLY when all the services are launched by docker
	// Poll until all the services are showed in docker-compose ps
	for len(containers) < len(podServices) {
		containers, err = GetContainerIdsByServices(files, services)
		if err != nil {
			logger.Errorln("Error retrieving container id list : ", err.Error())
			out <- types.POD_FAILED
			return
		}

		logger.Debugf("list of containers are launched : %v", containers)
		time.Sleep(interval)
	}

	logger.Println("Initial Health Check : Expected number of containers in monitoring : ", len(podServices))
	logger.Println("Initial Health Check : Actual number of containers in monitoring : ", len(containers))
	logger.Println("Container List : ", containers)

	// Get infra container id
	var systemProxyId string
	var hasInfra bool
	if _, hasInfra = podServices[types.INFRA_CONTAINER]; hasInfra {
		systemProxyId, err = GetContainerIdByService(files, types.INFRA_CONTAINER)
		if err != nil {
			logger.Errorf("Error getting container id of service %s: %v", types.INFRA_CONTAINER, err)
			log.Println("POD_INIT_HEALTH_CHECK_FAILURE -- Send Failed")
			out <- types.POD_FAILED
			return
		}
	}
	logger.Printf("Pod has infra container: %v", hasInfra)

healthCheck:
	for len(containers) != healthCount {
		healthCount = 0

		for i := 0; i < len(containers); i++ {

			var healthy string
			var exitCode int
			var running bool

			if hc, ok := HealthCheckListId[containers[i]]; ok && hc {
				healthy, running, exitCode, err = CheckContainer(containers[i], true)
			} else {
				if hc, err = isHealthCheckConfigured(containers[i]); hc {
					healthy, running, exitCode, err = CheckContainer(containers[i], true)
				} else {
					healthy, running, exitCode, err = CheckContainer(containers[i], false)
				}
			}

			if err != nil || healthy == types.UNHEALTHY {
				log.Println("POD_INIT_HEALTH_CHECK_FAILURE -- Send Failed")
				out <- types.POD_FAILED
				return
			}

			if healthy == types.HEALTHY {
				healthCount++
			}

			if exitCode == 0 && !running {
				log.Printf("Remove exited(exit code = 0)container %s from monitor list", containers[i])
				containers = append(containers[:i], containers[i+1:]...)
				i--
				healthCount--
			}

			// Break health check IF only system proxy is running
			if hasInfra && len(containers) == 1 && containers[0] == systemProxyId {
				break healthCheck
			}
		}

		if len(containers) != healthCount {
			time.Sleep(interval)
		}
	}

	PodContainers = make([]string, len(containers))
	copy(PodContainers, containers)

	logger.Printf("Health Check List: %v", HealthCheckListId)
	logger.Printf("Pod Monitor List: %v", PodContainers)

	isService := config.IsService()
	logger.Printf("Task is SERVICE: %v", isService)
	if len(containers) == 0 && !isService {
		logger.Println("Task is ADHOC job. Send POD_FINISHED")
		out <- types.POD_FINISHED
	} else if len(containers) == 0 && isService {
		logger.Println("Task is SERVICE. Send POD_FAILED")
		out <- types.POD_FAILED
	} else if !isService && hasInfra && len(containers) == 1 && containers[0] == systemProxyId {
		logger.Println("Task is ADHOC job. Only infra container is running, send POD_FINISHED")
		out <- types.POD_FINISHED
	} else if isService && hasInfra && len(containers) == 1 && containers[0] == systemProxyId {
		logger.Println("Task is SERVICE. Only infra container is running, send POD_FAILED")
		out <- types.POD_FAILED
	} else {
		logger.Println("Initial Health Check : send POD_RUNNING")
		out <- types.POD_RUNNING
	}

	logger.Println("====================Health check Done====================")

	GetPodDetail(files, "", false)
}

// check if primary container unable health check or not
func isHealthCheckConfigured(containerId string) (bool, error) {
	out, err := utils.RetryCmd(config.GetMaxRetry(), exec.Command("docker", "inspect", "--format='{{if .State.Health }}{{.State.Health.Status}}{{ end }}'", containerId))
	if err != nil {
		log.Errorf("Error executing cmd to check if healtcheck configured: %v", err)
		return false, err
	}

	_out := strings.Replace(strings.TrimSuffix(string(out[:]), "\n"), "'", "", -1)
	if _out == "" {
		return false, nil
	}

	HealthCheckListId[containerId] = true
	//log.Debugf("Initial Health Check : Container %s Health check is configured to true", containerId)
	return true, nil
}

func IsService(taskInfo *mesos.TaskInfo) bool {
	d := thrift.NewTDeserializer()
	assignTask := aurora.NewAssignedTask()
	d.Read(assignTask, taskInfo.GetData())
	return assignTask.Task.IsService
}
