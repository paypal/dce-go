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

package example

import (
	"context"

	"github.com/mesos/mesos-go/api/v0/executor"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/plugin"
	"github.com/paypal/dce-go/types"
	"github.com/paypal/dce-go/utils/pod"
	log "github.com/sirupsen/logrus"
)

var logger *log.Entry

type exampleExt struct {
}

func init() {
	log.SetOutput(config.CreateFileAppendMode(types.DCE_OUT))
	logger = log.WithFields(log.Fields{
		"plugin": "example",
	})

	logger.Println("Plugin Registering")

	plugin.ComposePlugins.Register(new(exampleExt), "example")
}

func (p *exampleExt) Name() string {
	return "example"
}

func (ex *exampleExt) LaunchTaskPreImagePull(ctx context.Context, composeFiles *[]string, executorId string, taskInfo *mesos.TaskInfo) error {
	logger.Println("LaunchTaskPreImagePull begin")
	// docker compose YML files are saved in context as type SERVICE_DETAIL which is map[interface{}]interface{}.
	// Massage YML files and save it in context.
	// Then pass to next plugin.

	// Get value from context
	filesMap := pod.GetServiceDetail()

	// Add label in each service, in each compose YML file
	for _, file := range *composeFiles {
		servMap := filesMap[file][types.SERVICES].(map[interface{}]interface{})
		for serviceName := range servMap {
			containerDetails := filesMap[file][types.SERVICES].(map[interface{}]interface{})[serviceName].(map[interface{}]interface{})
			if labels, ok := containerDetails[types.LABELS].(map[interface{}]interface{}); ok {
				labels["com.company.label"] = "awesome"
				containerDetails[types.LABELS] = labels
			}

		}

	}

	// Save the changes back to context
	pod.SetServiceDetail(filesMap)

	return nil
}

func (ex *exampleExt) LaunchTaskPostImagePull(ctx context.Context, composeFiles *[]string, executorId string, taskInfo *mesos.TaskInfo) error {
	logger.Println("LaunchTaskPostImagePull begin")
	return nil
}

func (ex *exampleExt) PostLaunchTask(ctx context.Context, composeFiles []string, taskInfo *mesos.TaskInfo) (string, error) {
	logger.Println("PostLaunchTask begin")
	return "", nil
}

func (ex *exampleExt) PreStopPod() error {
	logger.Println("PreStopPod Starting")
	return nil
}

func (ex *exampleExt) PreKillTask(ctx context.Context, taskInfo *mesos.TaskInfo) error {
	logger.Println("PreKillTask begin")
	return nil
}

func (ex *exampleExt) PostKillTask(ctx context.Context, taskInfo *mesos.TaskInfo) error {
	logger.Println("PostKillTask begin")
	return nil
}

func (ex *exampleExt) Shutdown(taskInfo *mesos.TaskInfo, ed executor.ExecutorDriver) error {
	logger.Println("Shutdown begin")
	return nil
}
