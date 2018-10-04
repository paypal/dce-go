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
