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

package config

import (
	"testing"

	"github.com/mesos/mesos-go/mesosproto"
	"github.com/stretchr/testify/assert"
)

func TestSetConfig(t *testing.T) {
	var val bool
	getConfigFromFile("config")
	SetConfig("test", true)
	val = GetConfig().GetBool("test")
	if !val {
		t.Errorf("expected val to be true, but got %v", val)
	}
}

func TestGetAppFolder(t *testing.T) {
	folder := GetAppFolder()
	if folder != "poddata" {
		t.Errorf("expected folder to be poddata, but got %s", folder)
	}
}

func TestGetStopTimeout(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int
	}{
		// this default is picked from the config.yaml file
		{"check default value", "", 20},
		{"incorrect integer value", 25, 0},
		{"incorrect string value", "25", 0},
		{"correct duration value", "25s", 25},
		{"correct duration value", "25m", 1500},
		{"incorrect value", "xyz", 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// this is to use the default value
			if test.input != "" {
				GetConfig().Set(COMPOSE_STOP_TIMEOUT, test.input)
			}

			got := GetStopTimeout()
			if got != test.want {
				t.Errorf("expected cleanpod.timeout to be %d for input %v, but got %d", test.want, test.input, got)
			}
		})
	}
}

func TestGetMaxRetry(t *testing.T) {
	max := GetMaxRetry()
	if max != 3 {
		t.Errorf("expected max retry to be 3, but got %d", max)
	}
}

func TestGetPullRetryCount(t *testing.T) {
	count := GetPullRetryCount()
	if count != 3 {
		t.Errorf("expected pull retry to be 3, but got %d", count)
	}
}

func TestOverrideConfig(t *testing.T) {
	type test struct {
		key         string
		val         string
		expectedKey string
		expectedVal string
		msg         string
	}

	overrideTests := []test{
		{"config.test1", "test1", "test1key", "", "shouldn't reset config if key isn't set"},
		{"config.cleanpod.timeout", "1", "cleanpod.timeout", "1", "should reset config if key is set"},
		{"config.launchtask.timeout", "1", "launchtask.timeout", "1", "should reset config if key is set"},
		{"config1.launchtask.timeout", "2", "launchtask.timeout", "1", "shouldn't reset config with invalid prefix"},
	}

	for _, ot := range overrideTests {
		var labels []*mesosproto.Label
		labels = append(labels, &mesosproto.Label{Key: &ot.key, Value: &ot.val})
		taskInfo := &mesosproto.TaskInfo{Labels: &mesosproto.Labels{Labels: labels}}
		OverrideConfig(taskInfo)
		assert.Equal(t, ot.expectedVal, GetConfig().GetString(ot.expectedKey), ot.msg)
	}
}
