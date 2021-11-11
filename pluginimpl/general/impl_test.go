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
	"os/exec"
	"strconv"
	"testing"

	"github.com/mesos/mesos-go/api/v0/mesosproto"
	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/types"
	"github.com/stretchr/testify/assert"
)

func TestCreateInfraContainer(t *testing.T) {
	config.GetConfig().SetDefault(types.NO_FOLDER, true)
	ctx := context.Background()
	_, err := CreateInfraContainer(ctx, "testdata/docker-infra-container.yml")
	if err != nil {
		t.Errorf("expected no error, but got %v", err)
	}
}

func TestGeneralExt_LaunchTaskPreImagePull(t *testing.T) {
	config.GetConfig().SetDefault(types.NO_FOLDER, true)
	g := new(generalExt)
	ctx := context.Background()
	begin, _ := strconv.ParseUint("1000", 10, 64)
	end, _ := strconv.ParseUint("1003", 10, 64)
	r := []*mesosproto.Value_Range{
		{Begin: &begin,
			End: &end},
	}
	ranges := &mesosproto.Value_Ranges{Range: r}
	name := types.PORTS
	taskInfo := &mesosproto.TaskInfo{
		Resources: []*mesosproto.Resource{
			{
				Name:   &name,
				Ranges: ranges,
			},
		},
	}

	// No compose file
	err := g.LaunchTaskPreImagePull(ctx, &[]string{}, "exeutorid", taskInfo)
	assert.Equal(t, string(types.NoComposeFile), err.Error(), "Test if compose file list is empty")

	// One compose file
	compose := []string{"testdata/test.yml"}
	err = g.LaunchTaskPreImagePull(ctx, &compose, "exeutorid", taskInfo)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(compose))
	exec.Command("rm", "docker-infra-container.yml").Run()
}
