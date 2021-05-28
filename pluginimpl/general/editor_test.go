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
	"strconv"
	"testing"

	"fmt"

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

	var extraHosts = make(map[interface{}]bool)
	ctx = context.WithValue(ctx, types.SERVICE_DETAIL, servDetail)
	_, curPort, _ := editComposeFile(ctx, "testdata/test.yml", "executorId", "taskId", ports.Front(), extraHosts)

	if curPort == nil || strconv.FormatUint(curPort.Value.(uint64), 10) != "3000" {
		t.Errorf("expected current port to be 3000 but got %v", curPort)
	}
	err = file.WriteChangeToFiles()
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

	var extraHosts = make(map[interface{}]bool)
	_, curPort, _ := editComposeFile(ctx, "testdata/test.yml", "executorId", "taskId", ports.Front(), extraHosts)

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
	if _, ok := containerDetailsAfter[types.EXTRA_HOSTS]; ok {
		t.Error("Expected extra hosts should be removed")
	}
	assert.Equal(t, 1, len(extraHosts), "test extra host")
}

func Test_scanForExtraHostsSection(t *testing.T) {
	var extraHosts = make(map[interface{}]bool)
	containerDetail := make(map[interface{}]interface{})
	containerDetail[types.EXTRA_HOSTS] = []interface{}{"redis1:0.0.0.0"}
	scanForExtraHostsSection(containerDetail, extraHosts)
	assert.Equal(t, 1, len(extraHosts), "Remove extra host from service")
	if _, ok := containerDetail[types.EXTRA_HOSTS]; ok {
		t.Errorf("Expected extra host is removed, but got %v\n", containerDetail[types.EXTRA_HOSTS])
	}

	scanForExtraHostsSection(containerDetail, extraHosts)
	assert.Equal(t, 1, len(extraHosts), "Remove extra host from service if no extra host is defined")
	if _, ok := containerDetail[types.EXTRA_HOSTS]; ok {
		t.Errorf("Expected extra host is removed, but got %v\n", containerDetail[types.EXTRA_HOSTS])
	}
}

func Test_addExtraHostsSection(t *testing.T) {
	config.GetConfig().SetDefault(types.NO_FOLDER, true)
	var servDetail types.ServiceDetail
	servDetail, err := file.ParseYamls(&[]string{"testdata/docker-extra-host.yml"})
	if err != nil {
		t.Fatalf("Error to parse yaml files : %v", err)
	}
	fmt.Println(servDetail)
	var ctx context.Context
	var extraHosts = make(map[interface{}]bool)
	ctx = context.Background()
	ctx = context.WithValue(ctx, types.SERVICE_DETAIL, servDetail)
	containerDetail := make(map[interface{}]interface{})
	containerDetail[types.EXTRA_HOSTS] = []interface{}{"redis2:0.0.0.2"}
	scanForExtraHostsSection(containerDetail, extraHosts)
	addExtraHostsSection(ctx, "testdata/docker-extra-host.yml", "redis", extraHosts)
	filesMap, ok := ctx.Value(types.SERVICE_DETAIL).(types.ServiceDetail)
	if !ok {
		t.Error("Couldn't get service detail")
		return
	}
	servMap, ok := filesMap["testdata/docker-extra-host.yml"][types.SERVICES].(map[interface{}]interface{})
	if !ok {
		t.Error("Couldn't get content of compose file ")
		return
	}

	if containerDetail, ok := servMap["redis"].(map[interface{}]interface{}); !ok {
		t.Error("Couldn't convert service to map[interface{}]interface{}")
	} else {
		res := containerDetail[types.EXTRA_HOSTS].([]interface{})
		assert.Equal(t, 3, len(res), "Adding extra host to service")
		fmt.Println("extra host:", res)
	}

	preCtx := ctx
	addExtraHostsSection(ctx, "testdata/docker-extra-host.yml", "fake", extraHosts)
	assert.Equal(t, preCtx, ctx, "Adding extra host to non exist service")
}
