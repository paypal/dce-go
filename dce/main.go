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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	exec "github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
	log "github.com/sirupsen/logrus"

	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/dce/monitor"
	"github.com/paypal/dce-go/plugin"
	_ "github.com/paypal/dce-go/plugin/example"
	_ "github.com/paypal/dce-go/plugin/general"
	"github.com/paypal/dce-go/types"
	"github.com/paypal/dce-go/utils"
	fileUtils "github.com/paypal/dce-go/utils/file"
	"github.com/paypal/dce-go/utils/pod"
	"github.com/paypal/dce-go/utils/wait"
)

var logger *log.Entry
var extpoints []plugin.ComposePlugin

type dockerComposeExecutor struct {
	tasksLaunched int
}

func newDockerComposeExecutor() *dockerComposeExecutor {
	return &dockerComposeExecutor{tasksLaunched: 0}
}

func (exec *dockerComposeExecutor) Registered(driver exec.ExecutorDriver, execInfo *mesos.ExecutorInfo, fwinfo *mesos.FrameworkInfo, slaveInfo *mesos.SlaveInfo) {
	log.Println("====================Mesos Registered====================")
	log.Println("Mesos Register : Registered Executor on slave ", slaveInfo.GetHostname())
	pod.ComposeExcutorDriver = driver
}

func (exec *dockerComposeExecutor) Reregistered(driver exec.ExecutorDriver, slaveInfo *mesos.SlaveInfo) {
	log.Println("====================Mesos Reregistered====================")
	log.Println("Mesos Re-registered Re-registered : Executor on slave ", slaveInfo.GetHostname())
}

func (exec *dockerComposeExecutor) Disconnected(exec.ExecutorDriver) {
	log.Println("====================Mesos Disconnected====================")
	log.Println("Mesos Disconnected : Mesos Executordisconnected.")
}

func (exec *dockerComposeExecutor) LaunchTask(driver exec.ExecutorDriver, taskInfo *mesos.TaskInfo) {
	log.SetOutput(config.CreateFileAppendMode(types.DCE_OUT))

	log.Println("====================Mesos LaunchTask====================")
	pod.ComposeExcutorDriver = driver
	logger = log.WithFields(log.Fields{
		"requuid":   pod.GetLabel("requuid", taskInfo),
		"tenant":    pod.GetLabel("tenant", taskInfo),
		"namespace": pod.GetLabel("namespace", taskInfo),
		"pool":      pod.GetLabel("pool", taskInfo),
	})

	task, err := json.Marshal(taskInfo)
	if err != nil {
		log.Println("Error marshalling taskInfo", err.Error())
	}
	buf := new(bytes.Buffer)
	json.Indent(buf, task, "", " ")
	logger.Debugln("taskInfo : ", buf)

	isService := pod.IsService(taskInfo)
	log.Printf("task is service: %v\n", isService)
	config.GetConfig().Set(types.IS_SERVICE, isService)

	pod.ComposeTaskInfo = taskInfo
	executorId := taskInfo.GetExecutor().GetExecutorId().GetValue()

	// Update pod status to STARTING
	pod.SetPodStatus(types.POD_STARTING)

	// Update mesos state TO STARTING
	pod.SendMesosStatus(driver, taskInfo.GetTaskId(), mesos.TaskState_TASK_STARTING.Enum())

	// Get required compose file list
	pod.ComposeFiles, _ = fileUtils.GetFiles(taskInfo)

	// Generate app folder to keep temp files
	err = fileUtils.GenerateAppFolder()
	if err != nil {
		logger.Errorln("Error creating app folder")
	}

	// Override config
	config.OverrideConfig(taskInfo)

	// Create context with timeout
	// Wait for pod launching until timeout
	var ctx context.Context
	var cancel context.CancelFunc
	ctx = context.Background()
	ctx, cancel = context.WithTimeout(ctx, config.GetLaunchTimeout()*time.Millisecond)
	go pod.WaitOnPod(&ctx)

	// Get order of plugins from config or mesos labels
	pluginOrder, err := fileUtils.GetPluginOrder(taskInfo)
	if err != nil {
		logger.Println("Plugin order missing in mesos label, trying to get it from config")
		pluginOrder = strings.Split(config.GetConfigSection("plugins")[types.PLUGIN_ORDER], ",")
	}
	pod.PluginOrder = pluginOrder
	logger.Println("PluginOrder : ", pluginOrder)

	// Select plugin extension points from plugin pools
	extpoints = plugin.GetOrderedExtpoints(pluginOrder)

	// Executing PreLaunchTask in order
	_, err = utils.PluginPanicHandler(utils.ConditionFunc(func() (string, error) {
		for i, ext := range extpoints {
			if ext == nil {
				logger.Errorln("Error getting plugins from plugin registration pools")
				return "", errors.New("plugin is nil")
			}
			err = ext.PreLaunchTask(&ctx, &pod.ComposeFiles, executorId, taskInfo)
			if err != nil {
				logger.Errorf("Error executing PreLaunchTask of plugin : %v\n", err)
				return "", err
			}
			if config.EnableComposeTrace() {
				fileUtils.DumpPluginModifiedComposeFiles(ctx, pluginOrder[i], i)
			}
		}
		return "", err
	}))
	if err != nil {
		pod.SetPodStatus(types.POD_FAILED)
		cancel()
		pod.SendMesosStatus(driver, taskInfo.GetTaskId(), mesos.TaskState_TASK_FAILED.Enum())
		return
	}

	// Service list from all compose files
	podServices := getServices(ctx)
	logger.Printf("pod service list: %v", podServices)

	// Write updated compose files into pod folder
	err = fileUtils.WriteChangeToFiles(ctx)
	if err != nil {
		logger.Errorf("Failure writing updated compose files : %v", err)
		pod.SetPodStatus(types.POD_FAILED)
		cancel()
		pod.SendMesosStatus(driver, taskInfo.GetTaskId(), mesos.TaskState_TASK_FAILED.Enum())
	}

	// Pulling image and launch pod
	replyPodStatus := pullAndLaunchPod()

	logger.Printf("Pod status returned by pullAndLaunchPod : %v", replyPodStatus)

	// Take an action depends on different status
	switch replyPodStatus {
	case types.POD_FAILED:
		cancel()
		pod.SendPodStatus(types.POD_FAILED)

	case types.POD_PULL_FAILED:
		cancel()
		pod.SendPodStatus(types.POD_PULL_FAILED)

	case types.POD_STARTING:
		// Initial health check
		res, err := initHealthCheck(podServices)
		if err != nil || res == types.POD_FAILED {
			cancel()
			pod.SendPodStatus(types.POD_FAILED)
		}

		// Temp status keeps the pod status returned by PostLaunchTask
		tempStatus, err := utils.PluginPanicHandler(utils.ConditionFunc(func() (string, error) {
			var tempStatus string
			for _, ext := range extpoints {
				logger.Println("Executing post launch task plugin")

				tempStatus, err = ext.PostLaunchTask(&ctx, pod.ComposeFiles, taskInfo)
				if err != nil {
					logger.Errorf("Error executing PostLaunchTask : %v", err)
				}
				logger.Printf("Get pod status : %s returned by PostLaunchTask", tempStatus)

				if tempStatus == types.POD_FAILED {
					return tempStatus, nil
				}
			}
			return tempStatus, nil
		}))
		if err != nil {
			logger.Errorf("Error executing PostLaunchTask : %v", err)
		}
		if tempStatus == types.POD_FAILED {
			cancel()
			pod.SendPodStatus(types.POD_FAILED)
		}

		if res == types.POD_RUNNING {
			cancel()
			if pod.GetPodStatus() != types.POD_RUNNING {
				pod.SendPodStatus(types.POD_RUNNING)
				go monitor.MonitorPoller()
			}
		}

		//For adhoc job, send finished to mesos if job already finished during init health check
		if res == types.POD_FINISHED {
			cancel()
			pod.SendPodStatus(types.POD_FINISHED)
		}

	default:
		logger.Printf("default: Unknown status -- %s from pullAndLaunchPod ", replyPodStatus)

	}

	logger.Println("====================Mesos LaunchTask Returned====================")
}

func (exec *dockerComposeExecutor) KillTask(driver exec.ExecutorDriver, taskId *mesos.TaskID) {
	log.Println("====================Mesos KillTask====================")

	logKill := log.WithFields(log.Fields{
		"taskId": taskId,
	})

	status := pod.GetPodStatus()
	switch status {
	case types.POD_FAILED:
		logKill.Printf("Mesos Kill Task : Current task status is %s , ignore killTask", status)

	case types.POD_RUNNING:
		logKill.Printf("Mesos Kill Task : Current task status is %s , continue killTask", status)
		pod.SetPodStatus(types.POD_KILLED)

		err := pod.StopPod(pod.ComposeFiles)
		if err != nil {
			logKill.Errorf("Error cleaning up pod : %v", err.Error())
		}

		err = pod.SendMesosStatus(driver, taskId, mesos.TaskState_TASK_KILLED.Enum())
		if err != nil {
			logKill.Errorf("Error during kill Task : %v", err.Error())
		}

		log.Println("====================Stop ExecutorDriver====================")
		time.Sleep(200 * time.Millisecond)
		driver.Stop()

	}

	logKill.Println("====================Mesos KillTask Stopped====================")
}

func (exec *dockerComposeExecutor) FrameworkMessage(driver exec.ExecutorDriver, msg string) {
	log.Printf("Got framework message: %v\n", msg)
}

func (exec *dockerComposeExecutor) Shutdown(driver exec.ExecutorDriver) {
	// Execute shutdown plugin extensions in order
	for _, ext := range extpoints {
		ext.Shutdown(pod.ComposeExcutorDriver)
	}
	log.Println("====================Stop ExecutorDriver====================")
	driver.Stop()
}

func (exec *dockerComposeExecutor) Error(driver exec.ExecutorDriver, err string) {
	log.Printf("Got error message : %v\n", err)
}

func pullAndLaunchPod() string {
	logger.Println("====================Pod Pull And Launch====================")

	if !config.SkipPullImages() {
		err := wait.PollRetry(config.GetPullRetryCount(), time.Duration(config.GetPollInterval())*time.Millisecond, wait.ConditionFunc(func() (string, error) {
			return "", pod.PullImage(pod.ComposeFiles)
		}))

		if err != nil {
			logger.Printf("POD_IMAGE_PULL_FAILED -- %v", err)
			return types.POD_PULL_FAILED
		}
	}

	return pod.LaunchPod(pod.ComposeFiles)
}

func initHealthCheck(podServices map[string]bool) (string, error) {
	res, err := wait.WaitUntil(config.GetLaunchTimeout()*time.Millisecond, wait.ConditionCHFunc(func(healthCheckReply chan string) {
		pod.HealthCheck(pod.ComposeFiles, podServices, healthCheckReply)
	}))

	if err != nil {
		log.Printf("POD_INIT_HEALTH_CHECK_TIMEOUT -- %v", err)
		return types.POD_FAILED, err
	}
	return res, err
}

func getServices(ctx context.Context) map[string]bool {
	podService := make(map[string]bool)
	filesMap := ctx.Value(types.SERVICE_DETAIL).(types.ServiceDetail)

	for _, file := range pod.ComposeFiles {
		servMap := filesMap[file][types.SERVICES].(map[interface{}]interface{})

		for serviceName := range servMap {
			podService[serviceName.(string)] = true
		}
	}
	return podService
}

func init() {
	flag.Parse()
	log.SetOutput(os.Stdout)

	// Set log to debug level when trace mode is turned on
	if config.EnableDebugMode() {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func main() {
	log.SetOutput(config.CreateFileAppendMode(types.DCE_OUT))

	log.Println("====================Genesis Executor (Go)====================")
	log.Println("created dce log file.")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGUSR1)
	go func() {
		for {
			sig := <-sig
			log.Printf("Received signal %s\n", sig.String())
			if sig == syscall.SIGUSR1 {
				switchDebugMode()
			}
		}
	}()

	dConfig := exec.DriverConfig{
		Executor: newDockerComposeExecutor(),
	}

	driver, err := exec.NewMesosExecutorDriver(dConfig)
	if err != nil {
		log.Errorf("Unable to create a ExecutorDriver : %v\n", err.Error())
	}

	_, err = driver.Start()
	if err != nil {
		log.Errorf("Got error: %v\n", err.Error())
		return
	}

	log.Println("Executor : Executor process has started and running.")
	driver.Join()

}

func switchDebugMode() {
	if config.EnableDebugMode() {
		config.GetConfig().Set(config.DEBUG_MODE, false)
		log.Println("###Turn off debug mode###")
		log.SetLevel(log.InfoLevel)
	} else {
		config.GetConfig().Set(config.DEBUG_MODE, true)
		log.Println("###Turn on debug mode###")
		log.SetLevel(log.DebugLevel)
	}
}
