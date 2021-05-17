package pod

import (
	"testing"

	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/stretchr/testify/assert"
)

func TestUpdateServiceDetail(t *testing.T) {
	// prepare taskInfo
	taskInfo := &mesos.TaskInfo{
		Labels: &mesos.Labels{},
	}

	s := GetServiceDetail(taskInfo)
	assert.Empty(t, s)

	s["a1"] = make(map[interface{}]interface{})
	s["a1"]["b1"] = "c1"
	s["a1"]["b2"] = 2

	err := UpdateServiceDetail(taskInfo, s)
	assert.NoError(t, err)

	s1 := GetServiceDetail(taskInfo)
	assert.EqualValues(t, s, s1)

}
