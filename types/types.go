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

package types

import (
	exec_cmd "os/exec"
	"sync"

	"github.com/mesos/mesos-go/mesosproto"
)

const (
	HEALTHY                 = "healthy"
	UNHEALTHY               = "unhealthy"
	STARTING                = "starting"
	POD_STAGING             = "POD_STAGING"
	POD_STARTING            = "POD_STARTING"
	POD_RUNNING             = "POD_RUNNING"
	POD_FAILED              = "POD_FAILED"
	POD_KILLED              = "POD_KILLED"
	POD_FINISHED            = "POD_FINISHED"
	POD_PULL_FAILED         = "PULL_IMAGE_FAILED"
	HEALTHCHECK             = "healthcheck"
	CONTAINER_NAME          = "container_name"
	NETWORK_MODE            = "network_mode"
	LINKS                   = "links"
	PORTS                   = "ports"
	LABELS                  = "labels"
	ENVIRONMENT             = "environment"
	RESTART                 = "restart"
	SERVICES                = "services"
	IMAGE                   = "image"
	VERSION                 = "version"
	NETWORKS                = "networks"
	TASK_ID                 = "taskId"
	EXECUTOR_ID             = "executorId"
	CGROUP_PARENT           = "cgroup_parent"
	PLUGIN_ORDER            = "pluginorder"
	INFRA_CONTAINER_YML     = "docker-infra-container.yml"
	HOST_MODE               = "host"
	NONE_NETWORK_MODE       = "none"
	NAME                    = "name"
	NETWORK_DRIVER          = "driver"
	NETWORK_DEFAULT_DRIVER  = "bridge"
	NETWORK_DEFAULT_NAME    = "default"
	NETWORK_EXTERNAL        = "external"
	INFRA_CONTAINER_GEN_YML = "docker-infra-container.yml-generated.yml"
	DEFAULT_FOLDER          = "poddata"
	NO_FOLDER               = "dontcreatefolder"
	RM_INFRA_CONTAINER      = "rm_infra_container"
	COMPOSE_HTTP_TIMEOUT    = "COMPOSE_HTTP_TIMEOUT"
	SERVICE_DETAIL          = "serviceDetail"
	INFRA_CONTAINER         = "networkproxy"
	IS_SERVICE              = "isService"
	FOREVER                 = 1<<63 - 1
)

type PodStatus struct {
	sync.RWMutex
	Status string
}

type ServiceDetail map[interface{}](map[interface{}]interface{})

type CmdResult struct {
	Result  error
	Command *exec_cmd.Cmd
}

type Network struct {
	PreExist bool
	Name     string
	Driver   string
}

type ContainerStatusDetails struct {
	ComposeTaskId *mesosproto.TaskID
	Pid           int
	ContainerId   string
	IsRunning     bool
	ExitCode      int
	HealthStatus  string
	RestartCount  int
	MaxRetryCount int
	Name          string
}

func (c *ContainerStatusDetails) SetContainerId(containerId string) {
	c.ContainerId = containerId
}

func (c *ContainerStatusDetails) SetComposeTaskId(composeTaskId *mesosproto.TaskID) {
	c.ComposeTaskId = composeTaskId
}
