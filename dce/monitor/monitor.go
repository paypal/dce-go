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
	var err error
	containers := make([]string, len(pod.PodContainers))
	copy(containers, pod.PodContainers)

	for i := 0; i < len(containers); i++ {
		var healthy string
		var exitCode int

		if hc, ok := pod.HealthCheckListId[containers[i]]; ok && hc {
			healthy, exitCode, err = pod.CheckContainer(containers[i], true)
			//log.Printf("container %s has health check, health status: %s, exitCode: %d, err : %v", containers[i], healthy, exitCode, err)
		} else {
			healthy, exitCode, err = pod.CheckContainer(containers[i], false)
			//log.Printf("container %s no health check, exitCode: %d, err : %v", containers[i], healthy, exitCode, err)
		}

		if err != nil {
			log.Println(fmt.Sprintf("Error to inspect container with id : %s, %v", containers[i], err.Error()))
			log.Println("Pod Monitor : Send Failed")
			return types.POD_FAILED
		}

		if exitCode != 0 && exitCode != -1 {
			log.Println("Pod Monitor : Stopped and send Failed")
			return types.POD_FAILED
		}

		if healthy == types.UNHEALTHY {
			if config.GetConfigSection(config.CLEANPOD) == nil ||
				config.GetConfigSection(config.CLEANPOD)[types.UNHEALTHY] == "true" {
				log.Println("Pod Monitor : Stopped and send Failed")
				return types.POD_FAILED
			}
			log.Warnf("Container %s became unhealthy", containers[i])
		}

		if exitCode == 0 || exitCode == -1 {
			containers = append(containers[:i], containers[i+1:]...)
			i--
		}

	}
	return ""
}

// Polling pod monitor periodically
func MonitorPoller() {
	log.Println("====================Pod Monitor Poller====================")

	gap, err := strconv.Atoi(config.GetConfigSection(config.LAUNCH_TASK)[config.POD_MONITOR_INTERVAL])
	if err != nil {
		log.Fatalf("Error to convert gap from string to int : %s|\n", err.Error())
	}

	res, _ := wait.PollForever(time.Duration(gap)*time.Millisecond, nil, wait.ConditionFunc(func() (string, error) {
		return podMonitor(), nil
	}))

	log.Printf("Pod Monitor Receiver : Received  message %s", res)

	curntPodStatus := pod.GetPodStatus()
	if curntPodStatus == types.POD_KILLED || curntPodStatus == types.POD_FAILED {
		log.Println("====================Pod Monitor Stopped====================")
		return
	}

	switch res {
	case types.POD_FAILED:
		pod.SendPodStatus(types.POD_FAILED)

	case types.POD_FINISHED:
		pod.SendPodStatus(types.POD_FINISHED)

	}

}
