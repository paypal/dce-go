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
	"strconv"
	"testing"

	"fmt"

	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/types"
	"github.com/paypal/dce-go/utils/file"
	"context"
)

func TestEditComposeFile(t *testing.T) {
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
	_, curPort, _ := EditComposeFile(&ctx, "testdata/test.yml", "executorId", "taskId", ports.Front())

	if curPort == nil || strconv.FormatUint(curPort.Value.(uint64), 10) != "3000" {
		t.Errorf("expected current port to be 3000 but got %v", curPort)
	}
	err = file.WriteChangeToFiles(ctx)
	if err != nil {
		t.Fatalf("Failed to write to files %v", err)
	}
}
