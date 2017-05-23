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
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/types"
	utils "github.com/paypal/dce-go/utils/wait"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	OUTPUT_DELIMITER        = ","
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

var ServiceNameMap = make(map[string]string)
var PodContainers []string

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

// Get set of containers id in pod
// docker-compose -f docker-compose.yaml ps -q
func GetPodContainers(files []string) ([]string, error) {
	var containers []string

	args := "docker-compose"
	for _, file := range files {
		args += " -f " + file
	}
	args += " ps | tail -n +3 | awk '{print $1}'"

	out, err := utils.RetryCmd(config.GetMaxRetry(), exec.Command("/bin/bash", "-c", args))
	if err != nil {
		log.Errorf("GetContainerIds : Error executing cmd docker-compose ps %#v", err)
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out[:])))
	for scanner.Scan() {
		containers = append(containers, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Errorln(os.Stderr, "reading standard input:", err)
	}
	return containers, nil
}

// Get set of running containers in pod
func GetRunningPodContainers(files []string) ([]string, error) {
	var containers []string

	args := "docker-compose"
	for _, file := range files {
		args += " -f " + file
	}
	args += " ps | grep Up | awk '{print $1}'"

	out, err := utils.RetryCmd(config.GetMaxRetry(), exec.Command("/bin/bash", "-c", args))
	if err != nil {
		log.Errorf("GetContainerIds : Error executing cmd docker-compose ps %#v", err)
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out[:])))
	for scanner.Scan() {
		containers = append(containers, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Errorln(os.Stderr, "reading standard input:", err)
	}
	return containers, nil
}

// docker-compose -f docker-compose.yaml ps
func GetPodDetail(files []string, primaryContainerId string, healthcheck bool) {
	parts, err := GenerateCmdParts(files, " ps")
	if err != nil {
		log.Errorln("Error generating cmd parts")
	}

	out, err := exec.Command("docker-compose", parts...).Output()
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

	parts, err := GenerateCmdParts(files, " up")
	if err != nil {
		log.Errorf("Error generating compose cmd parts : %s\n", err.Error())
		return types.POD_FAILED
	}

	log.Printf("Launch Pod : Command to launch task : docker-compose %v\n", parts)
	cmd := exec.Command("docker-compose", parts...)

	printStdout(cmd)
	printStderr(cmd)

	err = cmd.Start()
	if err != nil {
		log.Errorln("Error running launch task command : ", err.Error())
		return types.POD_FAILED
	}

	go func() {
		err = cmd.Wait()
		if err != nil {
			log.Errorln("Launch task cmd return non zero exit code : ", err.Error())
			SendPodStatus(types.POD_FAILED)
		}

		exitCode, err := CheckPodExitCode(files)
		if err != nil || exitCode != 0 {
			SendPodStatus(types.POD_FAILED)
		} else {
			SendPodStatus(types.POD_FINISHED)
		}
	}()

	return types.POD_STARTING
}

// Stop pod
// docker-compose stop
func StopPod(files []string) error {
	log.Println("====================Stop Pod====================")
	parts, err := GenerateCmdParts(files, " stop")
	if err != nil {
		log.Errorln("Error generating cmd parts : ", err.Error())
		return err
	}

	cmd := exec.Command("docker-compose", parts...)
	log.Println("Stop Pod : Command to stop task : docker-compose ", parts)

	err = cmd.Run()
	if err != nil {
		log.Errorln("Error stopping services :", err.Error())
		err = ForceKill(files)
		if err != nil {
			log.Errorf("Error in force pod kill : %v\n", err)
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

	networkName, err := GetContainerNetwork(config.GetConfig().GetString(types.INFRA_CONTAINER_NAME))
	if err != nil {
		log.Errorf("Failed to clean up network :%v", err)
	}
	err = RemoveNetwork(networkName)
	if err != nil {
		log.Errorf("Failed to clean up network :%v", err)
	}
	return nil
}

func GetContainerNetwork(name string) (string, error) {
	//cmd := exec.Command("docker", "inspect", "--format='{{.HostConfig.NetworkMode}}'", name)
	//out, err := utils.RetryCmd(config.GetMaxRetry(), cmd)

	out, err := exec.Command("docker", "inspect", "--format='{{.HostConfig.NetworkMode}}'", name).Output()
	if err != nil {
		log.Errorf("Error retrieving container network mode : %s , %s\n", name, err.Error())
		return "", err
	}

	network := strings.Replace(strings.TrimSuffix(string(out[:]), "\n"), "'", "", -1)
	log.Printf("Get container %s network : %s\n", name, network)
	return network, err
}

func RemoveNetwork(name string) error {
	log.Println("====================Remove network====================")
	cmd := exec.Command("docker", "network", "rm", name)
	//_, err := utils.RetryCmd(config.GetMaxRetry(), cmd)
	err := cmd.Run()
	if err != nil {
		log.Errorf("Error in rm network : %s , %s\n", name, err.Error())
	}
	return err
}

// Remove pod
// docker-compose down
func RemovePod(files []string) error {
	log.Println("====================Remove Pod====================")
	parts, err := GenerateCmdParts(files, " down")
	if err != nil {
		log.Errorln("Error generating compose cmd parts : ", err.Error())
		return err
	}

	cmd := exec.Command("docker-compose", parts...)
	log.Println("Remove Pod : Command to remove pod : docker-compose ", parts)

	err = cmd.Run()
	if err != nil {
		log.Errorln("Error in remove services :", err.Error())
		return err
	}
	return nil
}

// Force kill pod
// docker kill -f
func ForceKill(files []string) error {
	log.Println("====================Force Kill Pod====================")
	/*containerNames, err := GetRunningPodContainers(files)
	if err != nil {
		log.Errorln("Error to get container ids : ", err.Error())
	}
	for _, name := range containerNames {
		err = dockerKill(name)
		if err != nil {
			return err
		}
	}*/
	parts, err := GenerateCmdParts(files, " kill")
	if err != nil {
		log.Errorln("Error generating cmd parts : ", err.Error())
		return err
	}

	cmd := exec.Command("docker-compose", parts...)
	log.Println("Kill Pod : Command to kill task : docker-compose ", parts)

	err = cmd.Run()
	if err != nil {
		log.Errorln("Error in kill services :", err.Error())
		return err
	}
	return nil
}

// docker kill -f
func dockerKill(containerName string) error {
	cmd := exec.Command("docker", "kill", containerName)
	err := cmd.Run()
	if err != nil {
		log.Errorf("Error killing container : %s , %s\n", containerName, err.Error())
	}
	return err
}

// pull image
// docker-compose pull
func PullImage(files []string) error {
	log.Println("====================Pull Image====================")
	parts, err := GenerateCmdParts(files, " pull")
	if err != nil {
		log.Fatalf("Error generating cmd parts %s\n", err.Error())
		return err
	}

	cmd := exec.Command("docker-compose", parts...)
	log.Println("Pull Image : Command to pull images : docker-compose ", parts)

	printStdout(cmd)

	err = cmd.Start()
	if err != nil {
		log.Errorln("Error running pull images cmd: ", err.Error())
		return err
	}

	err = utils.WaitCmd(config.GetTimeout()*time.Millisecond, &types.CmdResult{
		Command: cmd,
	})
	if err != nil {
		return err
	}
	return nil
}

//Check container
//return healthy,run,err
func CheckContainer(containerId string, healthcheck bool) (string, int, error) {
	containerDetail, err := InspectContainerDetails(containerId, healthcheck)
	if err != nil {
		log.Errorf("CheckContainer : Error inspecting container with id : %s, %v", containerId, err.Error())
		return types.UNHEALTHY, 1, err
	}

	if containerDetail.ExitCode != 0 {
		log.Printf("CheckContainer : Container %s is finished with exit code %v\n", containerId, containerDetail.ExitCode)
		return types.UNHEALTHY, containerDetail.ExitCode, nil
	}

	if healthcheck {
		if containerDetail.IsRunning {
			//log.Printf("CheckContainer : Primary container %s is running , %s\n", containerId, containerDetail.HealthStatus)
			return containerDetail.HealthStatus, -1, nil
		}
		return containerDetail.HealthStatus, containerDetail.ExitCode, nil
	}

	if containerDetail.IsRunning {
		//log.Printf("CheckContainer : Regular container %s is running\n", containerId)
		return types.HEALTHY, -1, nil
	}

	return types.HEALTHY, 0, nil
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
		log.Errorf("Error inspecting container details : %v \n", err)
		return containerStatusDetails, err
	}

	containerStatusDetails, err = ParseToContainerDetail(string(out[:]), healthcheck)
	if err != nil {
		log.Errorf("Error parsing output to container details : %v \n", err)
		return containerStatusDetails, err
	}

	containerStatusDetails.SetContainerId(containerId)
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
	curntPodStatus := GetPodStatus()
	if curntPodStatus == types.POD_FAILED || curntPodStatus == types.POD_KILLED || curntPodStatus == status {
		log.Printf("Task has already been killed or failed or updated as required status: %s", curntPodStatus)
		return
	}

	SetPodStatus(status)

	switch status {
	case types.POD_RUNNING:
		SendMesosStatus(ComposeExcutorDriver, ComposeTaskInfo.GetTaskId(), mesos.TaskState_TASK_RUNNING.Enum())
	case types.POD_FINISHED:
		SendMesosStatus(ComposeExcutorDriver, ComposeTaskInfo.GetTaskId(), mesos.TaskState_TASK_FINISHED.Enum())
	case types.POD_FAILED:
		err := StopPod(ComposeFiles)
		if err != nil {
			log.Errorf("Error cleaning up pod : %v\n", err.Error())
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
		state.Enum().String() == mesos.TaskState_TASK_FINISHED.Enum().String() ||
		state.Enum().String() == mesos.TaskState_TASK_KILLED.Enum().String() {
		log.Println("====================Stop ExecutorDriver====================")
		driver.Stop()
	}
	return nil
}

func WaitOnPod(ctx *context.Context) {
	select {
	case <-(*ctx).Done():
		if (*ctx).Err() == context.DeadlineExceeded {
			log.Println("Timeout launching pod")
			SendPodStatus(types.POD_FAILED)
		} else if (*ctx).Err() == context.Canceled {
			log.Println("Stop wait on pod, since pod is running/finished/failed")
		}
	}
}

// print output of command
func printStdout(cmd *exec.Cmd) {
	log.Println("====================print stdout====================")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Errorln("Error in stdoutPipe : ", err.Error())
	}

	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			log.Println("Container Log  |", scanner.Text())
		}
	}()
}

// print error output of command
func printStderr(cmd *exec.Cmd) {
	log.Println("====================print stderr====================")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Errorln("Error in stderrPipe : ", err.Error())
	}

	scanner := bufio.NewScanner(stderr)
	go func() {
		for scanner.Scan() {
			log.Println("Container Log  |", scanner.Text())
		}
	}()
}

// healthCheck includes health checking for primary container and exit code checking for other containers
func HealthCheck(files []string, podServices map[string]bool, out chan<- string) {
	log.Println("====================Health Check====================", len(podServices))

	var err error
	var containers []string

	t, err := strconv.Atoi(config.GetConfigSection(config.LAUNCH_TASK)[config.POD_MONITOR_INTERVAL])
	if err != nil {
		log.Fatalf("Error converting interval time from string to int : %s\n", err.Error())
	}
	interval := time.Duration(t)

	for len(containers) < len(podServices) || !allServicesUp(containers, podServices) {
		containers, err = GetPodContainers(files)
		if err != nil {
			log.Errorln("Error retrieving container id list : ", err.Error())
			out <- types.POD_FAILED
			return
		}

		log.Printf("list of containers are launched : %v", containers)
		time.Sleep(interval)
	}

	PodContainers = make([]string, len(containers))
	copy(PodContainers, containers)

	log.Println("Initial Health Check : Expected number of containers in monitoring : ", len(podServices))
	log.Println("Initial Health Check : Acutal number of containers in monitoring : ", len(containers))
	log.Println("Container List : ", containers)

	// Keep health check until all the containers become healthy/running
	for len(containers) != 0 {

		for i := 0; i < len(containers); i++ {

			var healthy string

			if hc, ok := HealthCheckListId[containers[i]]; ok && hc || HealthCheckConfigured(containers[i]) {
				healthy, _, err = CheckContainer(containers[i], true)
			} else {
				healthy, _, err = CheckContainer(containers[i], false)
			}

			if err != nil || healthy == types.UNHEALTHY {
				log.Println("Initial Health Check : send FAILED")
				out <- types.POD_FAILED
				log.Println("Initial Health Check : send FAILED and stopped")
				return
			}

			if healthy == types.HEALTHY {
				// remove healthy container from check list
				log.Printf("Container %s is healthy, remove it from health check list", containers[i])
				containers = append(containers[:i], containers[i+1:]...)
				i--
			}
		}

		if len(containers) != 0 {
			time.Sleep(interval)
		}
	}

	log.Printf("Health Check List: %v", HealthCheckListId)

	log.Println("Initial Health Check : send POD_RUNNING")

	out <- types.POD_RUNNING

	log.Println("====================Health check Done====================")

	GetPodDetail(files, "", false)
}

// check if primary container unable health check or not
func HealthCheckConfigured(containerId string) bool {
	_, err := exec.Command("docker", "inspect", "--format='{{.State.Health.Status}}'", containerId).Output()
	if err != nil {
		//log.Printf("Initial Health Check : Container %s Health check is configured to false", containerId)
		return false
	} else {
		HealthCheckListId[containerId] = true
		//log.Printf("Initial Health Check : Contaienr %s Health check is configured to true", containerId)
		return true
	}
}

func allServicesUp(containers []string, podServices map[string]bool) bool {
	if len(containers) == 0 {
		return false
	}

	for _, container := range containers {
		var isService bool
		for service := range podServices {
			if strings.Contains(container, ServiceNameMap[service]) {
				isService = true
			}
		}
		if !isService {
			return false
		}
	}

	return true
}
