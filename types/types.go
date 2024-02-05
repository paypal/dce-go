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
	"strconv"

	"github.com/mesos/mesos-go/api/v0/mesosproto"
)

type PodStatus int

const (
	POD_STAGING PodStatus = 1 + iota
	POD_STARTING
	POD_RUNNING
	POD_FAILED
	POD_KILLED
	POD_FINISHED
	POD_PULL_FAILED
	POD_COMPOSE_CHECK_FAILED
	POD_EMPTY
)

func (status PodStatus) String() string {
	switch status {
	case POD_STAGING:
		return "POD_STAGING"
	case POD_STARTING:
		return "POD_STARTING"
	case POD_RUNNING:
		return "POD_RUNNING"
	case POD_FAILED:
		return "POD_FAILED"
	case POD_KILLED:
		return "POD_KILLED"
	case POD_FINISHED:
		return "POD_FINISHED"
	case POD_PULL_FAILED:
		return "POD_PULL_FAILED"
	case POD_COMPOSE_CHECK_FAILED:
		return "POD_COMPOSE_CHECK_FAILED"
	case POD_EMPTY:
		return ""
	}

	return ""
}

type HealthStatus int

const (
	STARTING HealthStatus = 1 + iota
	HEALTHY
	UNHEALTHY
	UNKNOWN_HEALTH_STATUS
)

func (status HealthStatus) String() string {
	switch status {
	case STARTING:
		return "starting"
	case HEALTHY:
		return "healthy"
	case UNHEALTHY:
		return "unhealthy"
	case UNKNOWN_HEALTH_STATUS:
		return "unknown"
	}

	return "unknown"
}

func GetInstanceStatusTag(svcContainer SvcContainer, healthy HealthStatus, running bool, exitCode int) map[string]string {
	return map[string]string{
		"serviceName":  svcContainer.ServiceName,
		"containerId":  svcContainer.ContainerId,
		"healthStatus": healthy.String(),
		"running":      strconv.FormatBool(running),
		"exitCode":     strconv.Itoa(exitCode),
	}
}

const (
	LOGLEVEL                = "loglevel"
	CONTAINER_NAME          = "container_name"
	NETWORK_MODE            = "network_mode"
	HEALTH_CHECK            = "healthcheck"
	LINKS                   = "links"
	PORTS                   = "ports"
	LABELS                  = "labels"
	ENVIRONMENT             = "environment"
	RESTART                 = "restart"
	APP_START_TIME          = "appStartTime"
	SERVICES                = "services"
	IMAGE                   = "image"
	VERSION                 = "version"
	NETWORKS                = "networks"
	HOSTNAME                = "hostname"
	VOLUMES                 = "volumes"
	DEPENDS_ON              = "depends_on"
	EXTRA_HOSTS             = "extra_hosts"
	CGROUP_PARENT           = "cgroup_parent"
	HOST_MODE               = "host"
	NONE_NETWORK_MODE       = "none"
	NAME                    = "name"
	NETWORK_DRIVER          = "driver"
	NETWORK_DEFAULT_DRIVER  = "bridge"
	NETWORK_DEFAULT_NAME    = "default"
	NETWORK_EXTERNAL        = "external"
	PLUGIN_ORDER            = "pluginorder"
	INFRA_CONTAINER_YML     = "docker-infra-container.yml"
	INFRA_CONTAINER_GEN_YML = "docker-infra-container.yml-generated.yml"
	DEFAULT_FOLDER          = "poddata"
	NO_FOLDER               = "dontcreatefolder"
	RM_INFRA_CONTAINER      = "rm_infra_container"
	COMPOSE_HTTP_TIMEOUT    = "COMPOSE_HTTP_TIMEOUT"
	SERVICE_DETAIL          = "serviceDetail"
	INFRA_CONTAINER         = "networkproxy"
	IS_SERVICE              = "isService"
	FOREVER                 = 1<<63 - 1
	DCE_OUT                 = "dce.out"
	DCE_ERR                 = "dce.err"
)

// ServiceDetail key is filepath, value is map to store Unmarshal the docker-compose.yaml
type ServiceDetail map[string]map[string]interface{}

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

type err string

const NoComposeFile err = "no compose file specified"

type StepData struct {
	RetryID    int               `json:"retryID,omitempty"`
	StepName   string            `json:"stepName,omitempty"`
	ErrorMsg   error             `json:"errorMsg,omitempty"`
	Status     string            `json:"status,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	StartTime  int64             `json:"startTime,omitempty"`
	EndTime    int64             `json:"endTime,omitempty"`
	ExecTimeMS int64             `json:"execTimeMS,omitempty"`
}

type SvcContainer struct {
	ServiceName string
	ContainerId string
	Pid         string
}
