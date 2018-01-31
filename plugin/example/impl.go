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
	"os"

	"context"

	"github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/paypal/dce-go/plugin"
	"github.com/paypal/dce-go/types"
	log "github.com/sirupsen/logrus"
)

var logger *log.Entry

type exampleExt struct {
}

func init() {
	logger = log.WithFields(log.Fields{
		"plugin": "example",
	})
	log.SetOutput(os.Stdout)

	logger.Println("Plugin Registering")

	plugin.ComposePlugins.Register(new(exampleExt), "example")
}

func (ex *exampleExt) PreLaunchTask(ctx *context.Context, composeFiles *[]string, executorId string, taskInfo *mesos.TaskInfo) error {
	logger.Println("PreLaunchTask begin")
	// docker compose YML files are saved in context as type SERVICE_DETAIL which is map[interface{}]interface{}.
	// Massage YML files and save it in context.
	// Then pass to next plugin.

	// Get value from context
	filesMap := (*ctx).Value(types.SERVICE_DETAIL).(types.ServiceDetail)

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
	*ctx = context.WithValue(*ctx, types.SERVICE_DETAIL, filesMap)

	return nil
}

func (ex *exampleExt) PostLaunchTask(ctx *context.Context, composeFiles []string, taskInfo *mesos.TaskInfo) (string, error) {
	logger.Println("PostLaunchTask begin")
	return "", nil
}

func (ex *exampleExt) PreKillTask(taskInfo *mesos.TaskInfo) error {
	logger.Println("PreKillTask begin")
	return nil
}

func (ex *exampleExt) PostKillTask(taskInfo *mesos.TaskInfo) error {
	logger.Println("PostKillTask begin")
	return nil
}

func (ex *exampleExt) Shutdown(executor.ExecutorDriver) error {
	logger.Println("Shutdown begin")
	return nil
}
