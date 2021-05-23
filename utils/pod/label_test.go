package pod

import (
	"testing"

	"gopkg.in/yaml.v2"

	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/stretchr/testify/assert"
)

func TestUpdateServiceDetail(t *testing.T) {
	// prepare taskInfo
	t.Run("manual-set", func(t *testing.T) {
		taskInfo := &mesos.TaskInfo{
			Labels: &mesos.Labels{},
		}

		s := GetServiceDetail(taskInfo)
		assert.Empty(t, s)

		s["a1"] = make(map[string]interface{})
		s["a1"]["b1"] = "c1"
		s["a1"]["b2"] = 2
		s["a1"]["b3"] = []interface{}{"c1, c2"}

		err := UpdateServiceDetail(taskInfo, s)
		assert.NoError(t, err)

		s1 := GetServiceDetail(taskInfo)
		assert.EqualValues(t, s, s1)
	})

	t.Run("parse-yaml", func(t *testing.T) {
		taskInfo := &mesos.TaskInfo{
			Labels: &mesos.Labels{},
		}

		m := make(map[string]interface{})
		sd := GetServiceDetail(taskInfo)

		var yamlFile = []byte(`
Hacker: true
name: steve
hobbies:
- skateboarding
- snowboarding
- go
clothing:
  jacket: leather
  trousers: denim
age: 35
eyes : brown
beard: true
`)

		err := yaml.Unmarshal(yamlFile, &m)
		assert.NoError(t, err)

		sd["f1"] = m

		err = UpdateServiceDetail(taskInfo, sd)
		assert.NoError(t, err)

		s1 := GetServiceDetail(taskInfo)
		assert.EqualValues(t, sd, s1)
	})

}
