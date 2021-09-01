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

package pod

import (
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/paypal/dce-go/plugin"
	"github.com/pkg/errors"

	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/types"
	"github.com/paypal/dce-go/utils/wait"
	"github.com/stretchr/testify/assert"
)

func TestLaunchPod(t *testing.T) {
	// file doesn't exist, should fail
	files := []string{"docker-fail.yml"}
	res, err := LaunchPod(files)
	assert.NoError(t, err)
	assert.EqualValues(t, res, types.POD_FAILED)

	// adhoc job
	files = []string{"testdata/docker-adhoc.yml"}
	res, err = LaunchPod(files)
	assert.NoError(t, err)
	assert.EqualValues(t, res, types.POD_STARTING)

	// long running job
	files = []string{"testdata/docker-long.yml"}
	res, err = LaunchPod(files)
	assert.NoError(t, err)
	assert.EqualValues(t, res, types.POD_STARTING)

	err = ForceKill(files)
	assert.NoError(t, err)
}

func TestGetContainerNetwork(t *testing.T) {
	// container exist
	network, err := GetContainerNetwork("redis_test")
	if err != nil {
		t.Errorf("expected no errors, but got %v", err)
	}
	if network != "testdata_default" {
		t.Errorf("expected network to be example_default, but got *%s*", network)
	}

	// container doesn't exist
	_, err = GetContainerNetwork("fake")
	if err == nil {
		t.Error("expected some errors here, but got nil")
	}

}

func TestRemoveNetwork(t *testing.T) {
	err := RemoveNetwork("testdata_default")
	if err != nil {
		t.Errorf("expected no error, but got %v", err)
	}
}

func TestForceKill(t *testing.T) {
	files := []string{"testdata/docker-long.yml"}
	res, err := LaunchPod(files)
	assert.NoError(t, err)
	if res != types.POD_STARTING {
		t.Fatalf("expected pod status to be POD_STARTING, but got %s", res)
	}
	err = ForceKill(files)
	assert.NoError(t, err)
}

func TestStopPod(t *testing.T) {
	files := []string{"testdata/docker-long.yml"}
	res, err := LaunchPod(files)
	assert.NoError(t, err)
	if res != types.POD_STARTING {
		t.Fatalf("expected pod status to be POD_STARTING, but got %s", res)
	}
	config.GetConfig().SetDefault(types.RM_INFRA_CONTAINER, true)
	err = StopPod(context.TODO(), files)
	assert.NoError(t, err)
	if err != nil {
		t.Errorf("expected no errors, but got %v", err)
	}
}

func TestGetContainerIdByService(t *testing.T) {
	files := []string{"testdata/docker-long.yml"}
	res, err := LaunchPod(files)
	if res != types.POD_STARTING {
		t.Fatalf("expected pod status to be POD_STARTING, but got %s", res)
	}

	res1, err := wait.PollUntil(time.Duration(1)*time.Second, nil, time.Duration(5)*time.Second, wait.ConditionFunc(func() (string, error) {
		return GetContainerIdByService(files, "redis")
	}))
	assert.NoError(t, err, "Test get container id should success")
	log.Println("Container id:", res1)

	res1, err = wait.PollUntil(time.Duration(1)*time.Second, nil, time.Duration(5)*time.Second, wait.ConditionFunc(func() (string, error) {
		return GetContainerIdByService(files, "fake")
	}))
	if err == nil {
		t.Fatal("Expected err isn't nil, but got nil")
	}
}

func TestKillContainer(t *testing.T) {
	err := KillContainer("", "")
	log.Println(err.Error())
	assert.Error(t, err, "test kill invalid container")

	files := []string{"testdata/docker-long.yml"}
	res, err := LaunchPod(files)
	assert.NoError(t, err)
	if res != types.POD_STARTING {
		t.Fatalf("expected pod status to be POD_STARTING, but got %s", res)
	}
	id, err := wait.PollUntil(time.Duration(1)*time.Second, nil, time.Duration(5)*time.Second, wait.ConditionFunc(func() (string, error) {
		return GetContainerIdByService(files, "redis")
	}))
	err = KillContainer("SIGUSR1", id)
	assert.NoError(t, err, "Test sending kill signal to container")
	err = KillContainer("", id)
	assert.NoError(t, err)

	config.GetConfig().Set(types.RM_INFRA_CONTAINER, true)
	err = StopPod(context.Background(), files)
	assert.NoError(t, err)
}

// TestGetLabel tests whether GetLabel()
// will break in case of an invalid key
func TestGetLabel(t *testing.T) {
	key1 := "org.apache.aurora.metadata.port"
	value1 := "9090"
	key2 := "org.apache.aurora.metadata.com.genesis.registrator.port"
	value2 := "9090"

	var labels []*mesos.Label
	labels = append(labels, &mesos.Label{
		Key:   &key1,
		Value: &value1,
	})
	labels = append(labels, &mesos.Label{
		Key:   &key2,
		Value: &value2,
	})

	taskInfo := &mesos.TaskInfo{
		Labels: &mesos.Labels{
			Labels: labels,
		},
	}
	type args struct {
		key      string
		taskInfo *mesos.TaskInfo
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "testSuccessShortLabel",
			args: args{
				key:      key1,
				taskInfo: taskInfo,
			},
			want: value1,
		},
		{
			name: "testSuccessLongLabel",
			args: args{
				key:      key2,
				taskInfo: taskInfo,
			},
			want: value2,
		},
		{
			name: "testOneWordKey",
			args: args{
				key:      "port",
				taskInfo: taskInfo,
			},
			want: "9090",
		}, {
			name: "testMultiWordsKey",
			args: args{
				key:      "registrator.port",
				taskInfo: taskInfo,
			},
			want: "9090",
		},
		{
			name: "testInvalidKey1",
			args: args{
				key:      "comINVALID.genesis.registrator.port",
				taskInfo: taskInfo,
			},
			want: "",
		},
		{
			name: "testInvalidKey2",
			args: args{
				key:      "INVALIDcom.genesis.registrator.port",
				taskInfo: taskInfo,
			},
			want: "",
		},
		{
			name: "testInvalidKey3",
			args: args{
				key:      "org.apache.aurora.metadata.INVALID",
				taskInfo: taskInfo,
			},
			want: "",
		},
		{
			name: "testInvalidKey4",
			args: args{
				key:      "org.apache.aurora.metadata",
				taskInfo: taskInfo,
			},
			want: "",
		},
		{
			name: "testInvalidKey5",
			args: args{
				key:      ".port",
				taskInfo: taskInfo,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetLabel(tt.args.key, taskInfo); got != tt.want {
				t.Errorf("GetLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAndRemoveLabel(t *testing.T) {
	key1 := "org.apache.aurora.metadata.nsvip"
	value1 := "1.1.1.1"
	key2 := "org.apache.aurora.metadata.com.paypal.apirouter.namespace"
	value2 := "namespace"
	var labels []*mesos.Label
	labels = append(labels, &mesos.Label{
		Key:   &key1,
		Value: &value1,
	})
	labels = append(labels, &mesos.Label{
		Key:   &key2,
		Value: &value2,
	})

	taskInfo := &mesos.TaskInfo{
		Labels: &mesos.Labels{
			Labels: labels,
		},
	}
	type args struct {
		key      string
		taskInfo *mesos.TaskInfo
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "testremovelabel",
			args: args{
				key:      key1,
				taskInfo: taskInfo,
			},
			want: value1,
		},
		{
			name: "testremovenonexistentlabel",
			args: args{
				key:      "invalidkey",
				taskInfo: taskInfo,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAndRemoveLabel(tt.args.key, tt.args.taskInfo); got != tt.want {
				t.Errorf("GetAndRemoveLabel() = %v, want %v", got, tt.want)

				labelsList := taskInfo.GetLabels().GetLabels()
				for _, label := range labelsList {
					if label.GetKey() == tt.args.key {
						t.Errorf("key %s not removed from tge taskInfo", tt.args.key)
					}
					if strings.Contains(label.GetKey(), tt.args.key) && strings.Contains(label.GetKey(), ".") {
						t.Errorf("key %s not removed from tge taskInfo", tt.args.key)
					}
				}
			}
		})
	}
}

func TestExecHooks(t *testing.T) {
	ctx := context.Background()
	//Register plugin with name
	if ok := plugin.PodStatusHooks.Register(&happyHook{}, "happyHook"); !ok {
		log.Fatalf("failed to register plugin %s", "happyHook")
	}

	if ok := plugin.PodStatusHooks.Register(&mandatoryHook{}, "mandatoryHook"); !ok {
		log.Fatalf("failed to register plugin %s", "mandatoryHook")
	}

	if ok := plugin.PodStatusHooks.Register(&panicHook{}, "panicHook"); !ok {
		log.Fatalf("failed to register plugin %s", "panicHook")
	}

	config.GetConfig().Set("podstatushooks.TASK_RUNNING", []string{"happyHook"})
	assert.NoError(t, execPodStatusHooks(ctx, "TASK_RUNNING", nil), "happy hook can't fail")

	config.GetConfig().Set("podstatushooks.TASK_FAILED", []string{"happyHook", "mandatoryHook"})
	assert.Error(t, execPodStatusHooks(ctx, "TASK_FAILED", nil), "mandatory hook can't succeed")

	config.GetConfig().Set("podstatushooks.TASK_FAILED", []string{"panicHook"})
	assert.Error(t, execPodStatusHooks(ctx, "TASK_FAILED", nil), "panicHook hook can't succeed")
}

// dummy executor hooks for unit test
type happyHook struct{}
type mandatoryHook struct{}
type panicHook struct{}

func (p *happyHook) Execute(ctx context.Context, podStatus string, data interface{}) (failExec bool, err error) {
	return true, nil
}

func (p *mandatoryHook) Execute(ctx context.Context, status string, data interface{}) (failExec bool, err error) {
	return true, errors.New("failure test case")
}

func (p *panicHook) Execute(ctx context.Context, status string, data interface{}) (failExec bool, err error) {
	panic("unit test panic")
	return false, errors.New("panic test case")
}
