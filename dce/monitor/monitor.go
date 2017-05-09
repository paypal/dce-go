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

	"strings"

	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/types"
	"github.com/paypal/dce-go/utils/pod"
	"github.com/paypal/dce-go/utils/wait"
)

// Watching pod status and notifying executor if any container in the pod goes wrong
func podMonitor(files []string) string {
	containers, err := pod.GetPodContainers(files)
	if err != nil {
		log.Errorln("Error retrieving container id list : ", err.Error())
		return types.POD_FAILED
	}

	for i := 0; i < len(containers); i++ {
		var healthy string
		var run bool

		/*if hc, ok := pod.HealthCheckList[containers[i]]; ok && hc || hasHealthCheck(containers[i]) {
			healthy, run, err = pod.CheckContainer(containers[i], true)
		} else {
			healthy, run, err = pod.CheckContainer(containers[i], false)
		}*/

		healthy, run, err = pod.CheckContainer(containers[i], false)

		if err != nil {
			log.Println(fmt.Sprintf("Error inspecting container with id : %s, %v", containers[i], err.Error()))
			log.Println("Pod Monitor : Send Failed")
			return types.POD_FAILED
		}

		if healthy == types.UNHEALTHY {
			log.Println("Pod Monitor : Stopped and send Failed")
			return types.POD_FAILED
		}

		if !run {
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
		log.Fatalf("Error converting podmonitorinterval from string to int : %s|\n", err.Error())
	}

	res, _ := wait.PollForever(time.Duration(gap)*time.Millisecond, nil, wait.ConditionFunc(func() (string, error) {
		return podMonitor(pod.ComposeFiles), nil
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

func hasHealthCheck(serviceName string) bool {
	for service := range pod.HealthCheckList {
		if strings.Contains(serviceName, service) {
			pod.HealthCheckList[serviceName] = true
			return true
		}
	}
	return false
}
