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

package general

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/types"
	utils "github.com/paypal/dce-go/utils/file"
	"github.com/paypal/dce-go/utils/pod"

	log "github.com/sirupsen/logrus"
)

const (
	PORT_DELIMITER = ":"
	PATH_DELIMITER = "/"
)

func EditComposeFile(ctx *context.Context, file string, executorId string, taskId string, ports *list.Element) (string, *list.Element, error) {
	var err error

	filesMap := (*ctx).Value(types.SERVICE_DETAIL).(types.ServiceDetail)
	if filesMap[file][types.SERVICES] == nil {
		log.Printf("Services is empty for file %s \n", file)
		return "", ports, nil
	}

	servMap, ok := filesMap[file][types.SERVICES].(map[interface{}]interface{})
	if !ok {
		log.Printf("Failed converting services to map[interface{}]interface{}")
		return "", ports, nil
	}

	for serviceName := range servMap {
		ports, err = UpdateServiceSessions(serviceName.(string), file, executorId, taskId, &filesMap, ports)
		if err != nil {
			log.Printf("Failed updating services: %v \n", err)
			return file, ports, err
		}
	}

	filesMap[file][types.VERSION] = "2.1"
	if !strings.Contains(file, utils.FILE_POSTFIX) {
		filesMap[file+utils.FILE_POSTFIX] = filesMap[file]
		delete(filesMap, file)
		file = file + utils.FILE_POSTFIX
	}
	*ctx = context.WithValue(*ctx, types.SERVICE_DETAIL, filesMap)

	logger.Printf("Updated compose files, current context: %v\n", filesMap)
	return file, ports, err
}

func UpdateServiceSessions(serviceName, file, executorId, taskId string, filesMap *types.ServiceDetail, ports *list.Element) (*list.Element, error) {
	containerDetails, ok := (*filesMap)[file][types.SERVICES].(map[interface{}]interface{})[serviceName].(map[interface{}]interface{})
	if !ok {
		log.Println("POD_UPDATE_YAML_FAIL")
	}
	logger := log.WithFields(log.Fields{
		"serviceName": serviceName,
		"taskId":      taskId,
	})

	// Remove restart session
	if _, ok := containerDetails[types.RESTART].(string); ok {
		delete(containerDetails, types.RESTART)
		log.Println("Edit Compose File : Remove restart")
	}

	// Get env list
	var envIsArray bool
	envMap, ok := containerDetails[types.ENVIRONMENT].(map[interface{}]interface{})
	if ok {
		logger.Printf("ENV is an array %v of %s : %v", envIsArray, serviceName, envMap)
	}
	envList, ok := containerDetails[types.ENVIRONMENT].([]interface{})
	if ok {
		logger.Printf("ENV is an array %v of %s : %v", envIsArray, serviceName, envList)
		envIsArray = true
	}
	if envMap == nil && envList == nil {
		envMap = make(map[interface{}]interface{})
	}

	if envIsArray {
		envList = append(envList, fmt.Sprintf("%s=%d", "PYTHONUNBUFFERED", 1))
		containerDetails[types.ENVIRONMENT] = envList
	} else {
		envMap["PYTHONUNBUFFERED"] = 1
		containerDetails[types.ENVIRONMENT] = envMap
	}

	// Update session of network_mode
	if serviceName != types.INFRA_CONTAINER {
		if network_mode, ok := containerDetails[types.NETWORK_MODE].(string); !ok ||
			(network_mode != types.HOST_MODE && network_mode != types.NONE_NETWORK_MODE) {

			containerDetails[types.NETWORK_MODE] = "service:" + types.INFRA_CONTAINER

		} else {
			config.GetConfig().SetDefault(types.RM_INFRA_CONTAINER, true)
		}

		logger.Println("Edit Compose File : update network mode")
	}

	// Update value of CONTAINER_NAME
	if containerName, ok := containerDetails[types.CONTAINER_NAME].(string); ok {
		containerName = utils.PrefixTaskId(taskId, containerName)
		containerDetails[types.CONTAINER_NAME] = containerName
		logger.Println("Edit Compose File : Updated container_name as ", containerName)
	}

	// Update value of LINKS
	if _, ok := containerDetails[types.LINKS].([]interface{}); ok {
		delete(containerDetails, types.LINKS)
	}

	// Tag containers in pod with taskId and executorId
	if labels, ok := containerDetails[types.LABELS].(map[interface{}]interface{}); !ok {
		if labels, ok := containerDetails[types.LABELS].([]interface{}); ok {
			labels = append(labels, fmt.Sprintf("%s=%s", types.EXECUTOR_ID, executorId))
			labels = append(labels, fmt.Sprintf("%s=%s", types.TASK_ID, taskId))
			containerDetails[types.LABELS] = labels
		} else {
			labels := make(map[interface{}]interface{})
			labels[types.TASK_ID] = taskId
			labels[types.EXECUTOR_ID] = executorId
			containerDetails[types.LABELS] = labels
		}
	} else {
		labels[types.TASK_ID] = taskId
		labels[types.EXECUTOR_ID] = executorId
		containerDetails[types.LABELS] = labels
	}

	logger.Println("Edit Compose File : Tag containers in pod with taskId and executorId ")

	// Add cgroup parent
	path, _ := filepath.Abs("")
	dirs := strings.Split(path, PATH_DELIMITER)
	containerDetails[types.CGROUP_PARENT] = "/mesos/" + dirs[len(dirs)-1]
	logger.Println("Edit Compose File : Add cgroup parent /mesos/", dirs[len(dirs)-1])

	// Update value of PORTS
	if networkMode, ok := containerDetails[types.NETWORK_MODE].(string); ok {

		if networkMode != types.HOST_MODE && networkMode != types.NONE_NETWORK_MODE {

			if portList, ok := containerDetails[types.PORTS].([]interface{}); ok {

				for i, p := range portList {
					portMap := strings.Split(p.(string), PORT_DELIMITER)
					if len(portMap) > 1 {
						if ports == nil {
							return nil, errors.New("No ports available")
						}
						p = strconv.FormatUint(ports.Value.(uint64), 10) + PORT_DELIMITER + portMap[1]
						portList[i] = p.(string)
						ports = ports.Next()
					} else {
						pod.SinglePort = true
					}
				}

				if strings.Contains(networkMode, "service:") {
					config_port := config.GetConfig().Get(types.PORTS)
					if config_port != nil {
						portList = append(config_port.([]interface{}), portList...)
					}
					config.GetConfig().Set(types.PORTS, portList)
					logger.Println("Set ports list : ", portList)
					delete(containerDetails, types.PORTS)
				} else {
					containerDetails[types.PORTS] = portList
				}

				logger.Println("Edit Compose File : Updated ports as ", portList)
			}
		}
	}

	// Add service ports to infra container
	if serviceName == types.INFRA_CONTAINER {
		if portList := config.GetConfig().Get(types.PORTS); portList != nil {

			logger.Println("Add services ports mapping to infra container ", portList)
			containerDetails[types.PORTS] = portList
		}
	}

	(*filesMap)[file][types.SERVICES].(map[interface{}]interface{})[serviceName] = containerDetails

	return ports, nil
}

func PostEditComposeFile(ctx *context.Context, file string) error {
	var err error
	filesMap := (*ctx).Value(types.SERVICE_DETAIL).(types.ServiceDetail)
	if filesMap[file][types.SERVICES] == nil {
		return nil
	}
	servMap := filesMap[file][types.SERVICES].(map[interface{}]interface{})
	for serviceName := range servMap {
		err = updateDynamicPorts(serviceName.(string), file, &filesMap)
		if err != nil {
			log.Errorf("Fail updating dynamic ports : %v", err)
			return err
		}
	}
	*ctx = context.WithValue(*ctx, types.SERVICE_DETAIL, filesMap)

	err = utils.WriteChangeToFiles(*ctx)
	if err != nil {
		log.Errorf("Failure writing updated compose files : %v", err)
		return err
	}
	return nil
}

func updateDynamicPorts(serviceName, file string, filesMap *types.ServiceDetail) error {
	containerDetails := (*filesMap)[file][types.SERVICES].(map[interface{}]interface{})[serviceName].(map[interface{}]interface{})
	ids, err := pod.GetPodContainerIds([]string{file})
	if err != nil {
		log.Errorf("Error retrieving infra container id : %v", err)
		return err
	}
	if portList, ok := containerDetails[types.PORTS].([]interface{}); ok {
		for i, p := range portList {
			portMap := strings.Split(p.(string), PORT_DELIMITER)
			if len(portMap) == 1 {
				dynamicPort, err := pod.GetDockerPorts(ids[0], portMap[0])
				if err != nil {
					log.Errorf("Error retrieving docker dynamic port : %v", err)
				}
				p = dynamicPort + PORT_DELIMITER + portMap[0]
				portList[i] = p.(string)
			}
		}

		containerDetails[types.PORTS] = portList
		(*filesMap)[file][types.SERVICES].(map[interface{}]interface{})[serviceName] = containerDetails

		logger.Println("Edit Compose File : Updated ports as ", portList)
	}
	return nil
}
