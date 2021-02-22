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

package http

import (
	"context"
	"encoding/json"
	"testing"
	"net/http"
)

type AllocateIPRequest struct {
	Label Label
}

type Label struct {
	Network     string
	ServiceType string
	ServiceName string
	Mobile      bool
	Id          string
}

func TestGetRequest(t *testing.T) {
	var ctx context.Context
	ctx = context.Background()
	tr := &http.Transport{
	}
	data, err := GetRequest(ctx, tr, "http://www.mocky.io/v2/589bb17d100000701266e5e1")
	if err != nil {
		t.Error(err.Error())
	}
	type AllocateIPReply struct {
		IpAddress        string
		LinkLocalAddress string
		Error            string
		XXX_unrecognized []byte `json:"-"`
	}
	var aip AllocateIPReply
	json.Unmarshal(data, &aip)
	if aip.IpAddress == "" || aip.LinkLocalAddress == "" {
		t.Fatal("expected IpAddress and LinkLocalAddress, but got empty")
	}
}

func TestGenBody(t *testing.T) {
	GenBody(AllocateIPRequest{
		Label{
			Network:     "webtier",
			ServiceType: "t2",
			ServiceName: "s3",
			Mobile:      true,
			Id:          "id",
		},
	})
}
