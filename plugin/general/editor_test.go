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
	"container/list"
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/types"
	"github.com/paypal/dce-go/utils/file"
	"github.com/stretchr/testify/assert"
)

func TestGenerateEditComposeFile(t *testing.T) {
	config.GetConfig().SetDefault(types.NO_FOLDER, true)
	var ports list.List
	u, _ := strconv.ParseUint("1000", 10, 64)
	ports.PushBack(u)
	u, _ = strconv.ParseUint("2000", 10, 64)
	ports.PushBack(u)
	u, _ = strconv.ParseUint("3000", 10, 64)
	ports.PushBack(u)

	var ctx context.Context
	ctx = context.Background()
	var servDetail types.ServiceDetail
	servDetail, err := file.ParseYamls(&[]string{"testdata/test.yml"})
	if err != nil {
		t.Fatalf("Error to parse yaml files : %v", err)
	}
	fmt.Println(servDetail)
	ctx = context.WithValue(ctx, types.SERVICE_DETAIL, servDetail)
	_, curPort, _ := editComposeFile(&ctx, "testdata/test.yml", "executorId", "taskId", ports.Front())

	if curPort == nil || strconv.FormatUint(curPort.Value.(uint64), 10) != "3000" {
		t.Errorf("expected current port to be 3000 but got %v", curPort)
	}
	err = file.WriteChangeToFiles(ctx)
	if err != nil {
		t.Fatalf("Failed to write to files %v", err)
	}
}

func Test_editComposeFile(t *testing.T) {
	config.GetConfig().SetDefault(types.NO_FOLDER, true)
	var ports list.List
	u, _ := strconv.ParseUint("1000", 10, 64)
	ports.PushBack(u)
	u, _ = strconv.ParseUint("2000", 10, 64)
	ports.PushBack(u)
	u, _ = strconv.ParseUint("3000", 10, 64)
	ports.PushBack(u)

	var ctx context.Context
	ctx = context.Background()
	var servDetail types.ServiceDetail
	servDetail, err := file.ParseYamls(&[]string{"testdata/test.yml"})
	if err != nil {
		t.Fatalf("Error to parse yaml files : %v", err)
	}
	ctx = context.WithValue(ctx, types.SERVICE_DETAIL, servDetail)

	// Before edit compose file
	containerDetails := servDetail["testdata/test.yml"][types.SERVICES].(map[interface{}]interface{})["mysql"].(map[interface{}]interface{})
	assert.Equal(t, nil, containerDetails[types.LABELS], "Before editing compose file")

	_, curPort, _ := editComposeFile(&ctx, "testdata/test.yml", "executorId", "taskId", ports.Front())

	// After edit compose file
	if curPort == nil || strconv.FormatUint(curPort.Value.(uint64), 10) != "3000" {
		t.Errorf("expected current port to be 3000 but got %v", curPort)
	}
	servDetailAfter := ctx.Value(types.SERVICE_DETAIL).(types.ServiceDetail)
	containerDetailsAfter := servDetailAfter["testdata/test.yml-generated.yml"][types.SERVICES].(map[interface{}]interface{})["mysql"].(map[interface{}]interface{})
	assert.Equal(t, 2, len(containerDetailsAfter[types.LABELS].(map[interface{}]interface{})), "After editing compose file")
	if _, ok := containerDetailsAfter[types.CGROUP_PARENT]; !ok {
		t.Error("Expected cgroup parent should be added")
	}
	if _, ok := containerDetailsAfter[types.RESTART]; ok {
		t.Error("Expected restart to be removed")
	}

}
