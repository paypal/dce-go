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
	"testing"

	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/types"
	"context"
)

func TestCreateInfraContainer(t *testing.T) {
	config.GetConfig().SetDefault(types.NO_FOLDER, true)
	var ctx context.Context
	ctx = context.Background()
	_, err := CreateInfraContainer(&ctx, "testdata/docker-infra-container.yml")
	if err != nil {
		t.Errorf("expected no error, but got %v", err)
	}
}
