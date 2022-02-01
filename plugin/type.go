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

//go:generate go-extpoints . ComposePlugin PodStatusHook Monitor
package plugin

import (
	"context"

	"github.com/mesos/mesos-go/api/v0/executor"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	"github.com/paypal/dce-go/types"
)

type ComposePlugin interface {
	// Name gets the name of the plugin
	Name() string

	// execute some tasks before the Image is pulled
	LaunchTaskPreImagePull(ctx context.Context, composeFiles *[]string, executorId string, taskInfo *mesos.TaskInfo) error

	// execute some tasks after the Image is pulled
	LaunchTaskPostImagePull(ctx context.Context, composeFiles *[]string, executorId string, taskInfo *mesos.TaskInfo) error

	// execute the tasks after the pod is launched
	PostLaunchTask(ctx context.Context, composeFiles []string, taskInfo *mesos.TaskInfo) (string, error)

	// execute the task before we send a Kill to Mesos
	PreKillTask(ctx context.Context, taskInfo *mesos.TaskInfo) error

	// execute the task after we send a Kill to Mesos
	PostKillTask(ctx context.Context, taskInfo *mesos.TaskInfo) error

	// execute the task to shutdown the pod
	Shutdown(taskInfo *mesos.TaskInfo, ed executor.ExecutorDriver) error
}

// PodStatusHook allows custom implementations to be plugged when a Pod (mesos task) status changes. Currently this is
// designed to be executed on task status changes during LaunchTask.
type PodStatusHook interface {
	// Execute is invoked when the pod.taskStatusCh channel has a new status. It returns an error on failure,
	// and also a flag "failExec" indicating if the error needs to fail the execution when a series of hooks are executed
	// This is to support cases where a few hooks can be executed in a best effort manner and need not fail the executor
	Execute(ctx context.Context, podStatus string, data interface{}) (failExec bool, err error)

	// This will be called from the pod.stopDriver. This will be used to end anything which needs clean-up at the end
	// of the executor
	Shutdown(ctx context.Context, podStatus string, data interface{})
}

// Monitor inspects pods periodically until pod failed or terminated It also defines when to consider a pod as failed.
// Move monitor as a plugin provides flexibility to replace default monitor logic.
// Monitor name presents in config `monitorName` will be used, otherwise, default monitor will be used.
type Monitor interface {
	Start(ctx context.Context) (types.PodStatus, error)
}
