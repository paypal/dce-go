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
	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/types"
)

/*func TestLaunchPod(t *testing.T) {
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

	err := RemovePod(files)
	if err != nil {
		t.Error(err.Error())
	}

}

func TestGetContainerNetwork(t *testing.T) {

	// container exist
	network, err := GetContainerNetwork("example_redis_1")
	if err != nil {
		t.Errorf("expected no errors, but got %v", err)
	}
	if network != "example_default" {
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
}*/

func TestGetRunningPodContainers(t *testing.T) {
	ids, err := GetRunningPodContainers([]string{"testdata/docker-long.yml"})
	if err != nil {
		t.Errorf("expected no errors, but got %v", err)
	}
	if len(ids) != 1 && len(ids) != 0 {
		t.Errorf("expected one/no running containers, but got %d", len(ids))
	}

}

func TestForceKill(t *testing.T) {
	err := ForceKill([]string{"testdata/docker-long.yml"})
	if err != nil {
		t.Errorf("expected no errors, but got %v", err)
	}
}

func TestStopPod(t *testing.T) {
	config.GetConfig().SetDefault(types.RM_INFRA_CONTAINER, true)
	err := StopPod([]string{"testdata/docker-long.yml"})
	if err != nil {
		t.Errorf("expected no errors, but got %v", err)
	}
}
