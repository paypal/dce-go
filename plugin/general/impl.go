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
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/plugin"
	"github.com/paypal/dce-go/types"
	utils "github.com/paypal/dce-go/utils/file"
	"github.com/paypal/dce-go/utils/pod"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var logger *log.Entry

type generalExt struct {
}

var infraYmlPath string

func init() {
	log.SetOutput(config.CreateFileAppendMode(types.DCE_OUT))

	logger = log.WithFields(log.Fields{
		"plugin": "general",
	})
	logger.Println("Plugin Registering")

	//Register plugin with name
	plugin.ComposePlugins.Register(new(generalExt), "general")

	//Merge plugin config file
	config.ConfigInit(utils.SearchFile(".", "general.yaml"))
}

func (ge *generalExt) PreLaunchTask(ctx *context.Context, composeFiles *[]string, executorId string, taskInfo *mesos.TaskInfo) error {
	logger.Println("PreLaunchTask begin")

	if composeFiles == nil || len(*composeFiles) == 0 {
		return fmt.Errorf(string(types.NoComposeFile))
	}

	var editedFiles []string
	var err error

	logger.Println("====================context in====================")
	logger.Println((*ctx).Value(types.SERVICE_DETAIL))

	logger.Printf("Current compose files list: %v", *composeFiles)

	if (*ctx).Value(types.SERVICE_DETAIL) == nil {
		var servDetail types.ServiceDetail
		servDetail, err = utils.ParseYamls(composeFiles)
		if err != nil {
			log.Errorf("Error parsing yaml files : %v", err)
			return err
		}

		*ctx = context.WithValue(*ctx, types.SERVICE_DETAIL, servDetail)
	}

	currentPort := pod.GetPorts(taskInfo)

	// Create infra container yml file
	infrayml, err := CreateInfraContainer(ctx, types.INFRA_CONTAINER_YML)
	if err != nil {
		logger.Errorln("Error creating infra container : ", err.Error())
		return err
	}
	*composeFiles = append(utils.FolderPath(*composeFiles), infrayml)

	var extraHosts = make(map[interface{}]bool)

	var indexInfra int
	for i, file := range *composeFiles {
		logger.Printf("Starting Edit compose file %s", file)
		var editedFile string
		editedFile, currentPort, err = editComposeFile(ctx, file, executorId, taskInfo.GetTaskId().GetValue(), currentPort, extraHosts)
		if err != nil {
			logger.Errorln("Error editing compose file : ", err.Error())
			return err
		}

		if strings.Contains(editedFile, types.INFRA_CONTAINER_GEN_YML) {
			indexInfra = i
			infraYmlPath = editedFile
		}

		if editedFile != "" {
			editedFiles = append(editedFiles, editedFile)
		}
	}

	// Remove infra container yml file if network mode is host
	if config.GetConfig().GetBool(types.RM_INFRA_CONTAINER) {
		logger.Printf("Remove file: %s\n", types.INFRA_CONTAINER_GEN_YML)
		filesMap := (*ctx).Value(types.SERVICE_DETAIL).(types.ServiceDetail)
		delete(filesMap, editedFiles[indexInfra])
		*ctx = context.WithValue(*ctx, types.SERVICE_DETAIL, filesMap)
		editedFiles = append(editedFiles[:indexInfra], editedFiles[indexInfra+1:]...)
		err = utils.DeleteFile(types.INFRA_CONTAINER_YML)
		if err != nil {
			log.Errorf("Error deleting infra yml file %v", err)
		}
	} else {
		// Move extra_hosts from other compose files to infra container
		addExtraHostsSection(ctx, infraYmlPath, types.INFRA_CONTAINER, extraHosts)
	}

	logger.Println("====================context out====================")
	logger.Println((*ctx).Value(types.SERVICE_DETAIL))

	*composeFiles = editedFiles

	logger.Println("Return massaged file list (Required Compose File List), ", *composeFiles)

	if currentPort != nil {
		logger.Println("Current port is ", strconv.FormatUint(currentPort.Value.(uint64), 10))
	}

	return nil
}

func (gp *generalExt) PostLaunchTask(ctx *context.Context, files []string, taskInfo *mesos.TaskInfo) (string, error) {
	logger.Println("PostLaunchTask begin")
	if pod.SinglePort {
		err := postEditComposeFile(ctx, infraYmlPath)
		if err != nil {
			log.Errorf("PostLaunchTask: Error editing compose file : %v", err)
			return types.POD_FAILED, err
		}
	}
	return "", nil
}

func (gp *generalExt) PreKillTask(taskInfo *mesos.TaskInfo) error {
	logger.Println("PreKillTask begin")
	return nil
}

// PostKillTask cleans up containers, volumes, images if task is killed by mesos
// Failed tasks will be cleaned up based on config cleanpod.cleanvolumeandcontaineronmesoskill and cleanpod.cleanimageonmesoskill
// Non pre-existing networks will always be removed
func (gp *generalExt) PostKillTask(taskInfo *mesos.TaskInfo) error {
	logger.Println("PostKillTask begin")
	var err error
	if pod.GetPodStatus() != types.POD_FAILED || (pod.GetPodStatus() == types.POD_FAILED && config.GetConfig().GetBool(config.CLEAN_FAIL_TASK)) {
		// clean pod volume and container if clean_container_volume_on_kill is true
		cleanVolumeAndContainer := config.GetConfig().GetBool(config.CLEAN_CONTAINER_VOLUME_ON_MESOS_KILL)
		if cleanVolumeAndContainer{
			err = pod.RemovePodVolume(pod.ComposeFiles)
			if err != nil {
				logger.Errorf("Error cleaning volumes: %v", err)
			}
		}

		// clean pod images if clean_image_on_kill is true
		cleanImage := config.GetConfig().GetBool(config.CLEAN_IMAGE_ON_MESOS_KILL)
		if cleanImage {
			err = pod.RemovePodImage(pod.ComposeFiles)
			if err != nil {
				logger.Errorf("Error cleaning images: %v", err)
			}
		}
	} else {
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
		infraContainerId, err := pod.GetContainerIdByService(pod.ComposeFiles, types.INFRA_CONTAINER)
		if err != nil {
			logger.Errorf("Error getting container id of service %s: %v", types.INFRA_CONTAINER, err)
			return nil
		}

		networkName, err := pod.GetContainerNetwork(infraContainerId)
		if err != nil {
			logger.Errorf("Failed to clean up network :%v", err)
		}
		err = pod.RemoveNetwork(networkName)
		if err != nil {
			logger.Errorf("POD_CLEAN_NETWORK_FAIL -- %v", err)
		}
	}
	return err
}

func (gp *generalExt) Shutdown(executor.ExecutorDriver) error {
	logger.Println("Shutdown begin")
	return nil
}

func CreateInfraContainer(ctx *context.Context, path string) (string, error) {
	containerDetail := make(map[interface{}]interface{})
	service := make(map[interface{}]interface{})
	_yaml := make(map[interface{}]interface{})

	containerDetail[types.CONTAINER_NAME] = config.GetConfigSection(config.INFRA_CONTAINER)[types.CONTAINER_NAME]
	containerDetail[types.IMAGE] = config.GetConfigSection(config.INFRA_CONTAINER)[types.IMAGE]

	if network, ok := config.GetNetwork(); ok {
		if network.PreExist {
			serviceNetworks := make(map[string]interface{})
			external := make(map[string]interface{})
			name := make(map[string]string)
			if network.Name == "" {
				log.Warningln("Error in configuration file! Network Name is required if PreExist is true")
				return "", errors.New("NetworkName missing in general.yaml")
			}
			name[types.NAME] = network.Name
			external[types.NETWORK_EXTERNAL] = name
			serviceNetworks[types.NETWORK_DEFAULT_NAME] = external
			_yaml[types.NETWORKS] = serviceNetworks

		} else {
			serviceNetworks := make(map[string]interface{})
			driver := make(map[string]string)
			if network.Driver == "" {
				network.Driver = types.NETWORK_DEFAULT_DRIVER
			}
			if network.Name == "" {
				network.Name = types.NETWORK_DEFAULT_NAME
			}
			driver[types.NETWORK_DRIVER] = network.Driver
			serviceNetworks[network.Name] = driver
			_yaml[types.NETWORKS] = serviceNetworks
			containerDetail[types.NETWORKS] = []string{network.Name}

		}
	}

	service[types.INFRA_CONTAINER] = containerDetail
	_yaml[types.SERVICES] = service
	_yaml[types.VERSION] = "2.1"
	log.Println(_yaml)

	content, _ := yaml.Marshal(_yaml)
	fileName, err := utils.WriteToFile(path, content)
	if err != nil {
		log.Errorf("Error writing infra container details into fils %v", err)
		return "", err
	}

	fileMap, ok := (*ctx).Value(types.SERVICE_DETAIL).(types.ServiceDetail)
	if !ok {
		log.Warningln("SERVICE_DETAIL missing in context value")
		fileMap = types.ServiceDetail{}
	}

	fileMap[fileName] = _yaml
	*ctx = context.WithValue(*ctx, types.SERVICE_DETAIL, fileMap)
	return fileName, nil
}
