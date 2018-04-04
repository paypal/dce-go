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
	"testing"

	//"github.com/paypal/dce-go/types"
	"time"

	"fmt"

	"github.com/mesos/mesos-go/examples/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/types"
	"github.com/paypal/dce-go/utils/wait"
)

func TestLaunchPod(t *testing.T) {
	// file doesn't exist, should fail
	files := []string{"docker-fail.yml"}
	res := LaunchPod(files)
	if res != types.POD_FAILED {
		t.Fatalf("expected pod status to be POD_FAILED, but got %s", res)
	}

	// adhoc job
	files = []string{"testdata/docker-adhoc.yml"}
	res = LaunchPod(files)
	if res != types.POD_STARTING {
		t.Fatalf("expected pod status to be POD_STARTING, but got %s", res)
	}

	// long running job
	files = []string{"testdata/docker-long.yml"}
	res = LaunchPod(files)
	if res != types.POD_STARTING {
		t.Fatalf("expected pod status to be POD_STARTING, but got %s", res)
	}
	err := ForceKill(files)
	if err != nil {
		t.Errorf("expected no errors, but got %v", err)
	}
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
	res := LaunchPod(files)
	if res != types.POD_STARTING {
		t.Fatalf("expected pod status to be POD_STARTING, but got %s", res)
	}
	err := ForceKill(files)
	if err != nil {
		t.Errorf("expected no errors, but got %v", err)
	}
}

func TestStopPod(t *testing.T) {
	files := []string{"testdata/docker-long.yml"}
	res := LaunchPod(files)
	if res != types.POD_STARTING {
		t.Fatalf("expected pod status to be POD_STARTING, but got %s", res)
	}
	config.GetConfig().SetDefault(types.RM_INFRA_CONTAINER, true)
	err := StopPod(files)
	if err != nil {
		t.Errorf("expected no errors, but got %v", err)
	}
}

func TestGetContainerIdByService(t *testing.T) {
	files := []string{"testdata/docker-long.yml"}
	res := LaunchPod(files)
	if res != types.POD_STARTING {
		t.Fatalf("expected pod status to be POD_STARTING, but got %s", res)
	}

	res, err := wait.PollUntil(time.Duration(1)*time.Second, nil, time.Duration(5)*time.Second, wait.ConditionFunc(func() (string, error) {
		return GetContainerIdByService(files, "redis")
	}))
	assert.NoError(t, err, "Test get container id should success")
	fmt.Println("Container id:", res)

	res, err = wait.PollUntil(time.Duration(1)*time.Second, nil, time.Duration(5)*time.Second, wait.ConditionFunc(func() (string, error) {
		return GetContainerIdByService(files, "fake")
	}))
	if err == nil {
		t.Fatal("Expected err isn't nil, but got nil")
	}
}
