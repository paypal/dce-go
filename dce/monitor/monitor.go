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

//Monitor pod status
package monitor

import (
	log "github.com/sirupsen/logrus"

	"fmt"

	"strconv"
	"time"

	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/types"
	"github.com/paypal/dce-go/utils/pod"
	"github.com/paypal/dce-go/utils/wait"
)

// Watching pod status and notifying executor if any container in the pod goes wrong
func podMonitor() string {
	logger := log.WithFields(log.Fields{
		"func": "monitor.podMonitor",
	})

	var err error

	for i := 0; i < len(pod.PodContainers); i++ {
		var healthy string
		var exitCode int

		if hc, ok := pod.HealthCheckListId[pod.PodContainers[i]]; ok && hc {
			healthy, exitCode, err = pod.CheckContainer(pod.PodContainers[i], true)
			logger.Debugf("container %s has health check, health status: %s, exitCode: %d, err : %v", pod.PodContainers[i], healthy, exitCode, err)
		} else {
			healthy, exitCode, err = pod.CheckContainer(pod.PodContainers[i], false)
			log.Debugf("container %s no health check, exitCode: %d, err : %v", pod.PodContainers[i], healthy, exitCode, err)
		}

		if err != nil {
			logger.Errorf(fmt.Sprintf("POD_MONITOR_HEALTH_CHECK_FAILED -- Error inspecting container with id : %s, %v", pod.PodContainers[i], err.Error()))
			logger.Errorln("POD_MONITOR_FAILED -- Send Failed")
			return types.POD_FAILED
		}

		if exitCode != 0 && exitCode != -1 {
			logger.Println("POD_MONITOR_APP_EXIT -- Stop pod monitor and send Failed")
			return types.POD_FAILED
		}

		if healthy == types.UNHEALTHY {
			if config.GetConfigSection(config.CLEANPOD) == nil ||
				config.GetConfigSection(config.CLEANPOD)[types.UNHEALTHY] == "true" {
				logger.Println("POD_MONITOR_HEALTH_CHECK_FAILED -- Stop pod monitor and send Failed")
				return types.POD_FAILED
			}
			logger.Warnf("Container %s became unhealthy, but pod won't be killed due to cleanpod config", pod.PodContainers[i])
		}

		if exitCode == 0 {
			logger.Printf("Removed finished(exit with 0) container %s from monitor list", pod.PodContainers[i])
			pod.PodContainers = append(pod.PodContainers[:i], pod.PodContainers[i+1:]...)
			i--

		}
	}

	if len(pod.PodContainers) == 0 {
		logger.Println("Pod Monitor : Finished")
		return types.POD_FINISHED
	}

	return ""
}

// Polling pod monitor periodically
func MonitorPoller() {
	logger := log.WithFields(log.Fields{
		"func": "monitor.MonitorPoller",
	})

	logger.Println("====================Pod Monitor Poller====================")

	gap, err := strconv.Atoi(config.GetConfigSection(config.LAUNCH_TASK)[config.POD_MONITOR_INTERVAL])
	if err != nil {
		logger.Errorf("Error converting podmonitorinterval from string to int : %s", err.Error())
		gap = 10000
	}

	res, err := wait.PollForever(time.Duration(gap)*time.Millisecond, nil, wait.ConditionFunc(func() (string, error) {
		return podMonitor(), nil
	}))

	logger.Printf("Pod Monitor Receiver : Received  message %s", res)

	curntPodStatus := pod.GetPodStatus()
	if curntPodStatus == types.POD_KILLED || curntPodStatus == types.POD_FAILED {
		logger.Println("====================Pod Monitor Stopped====================")
		return
	}

	if err != nil {
		pod.SendPodStatus(types.POD_FAILED)
		return
	}

	switch res {
	case types.POD_FAILED:
		pod.SendPodStatus(types.POD_FAILED)

	case types.POD_FINISHED:
		pod.SendPodStatus(types.POD_FINISHED)
	}
}
