package pod

import (
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/stretchr/testify/assert"
)

func TestUpdateServiceDetail(t *testing.T) {
	// prepare taskInfo
	t.Run("manual-set", func(t *testing.T) {
		s := GetServiceDetail()
		assert.Empty(t, s)

		s["a1"] = make(map[string]interface{})
		s["a1"]["b1"] = "c1"
		s["a1"]["b2"] = 2
		s["a1"]["b3"] = []interface{}{"c1, c2"}

		SetServiceDetail(s)

		s1 := GetServiceDetail()
		assert.EqualValues(t, s, s1)
	})

	t.Run("parse-yaml", func(t *testing.T) {

		m := make(map[string]interface{})
		sd := GetServiceDetail()

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

		SetServiceDetail(sd)

		s1 := GetServiceDetail()
		assert.EqualValues(t, sd, s1)
	})

}
