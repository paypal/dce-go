package pod

import (
	"errors"
	"fmt"
	"testing"

	"github.com/paypal/dce-go/types"
	"github.com/stretchr/testify/assert"
)

func TestPluginPanicHandler(t *testing.T) {
	_, err := PluginPanicHandler(ConditionFunc(func() (string, error) {
		panic("Test panic error")
	}))
	if err == nil {
		t.Error("Expected err not be nil, but got nil")
	}
}

func TestSetStepData(t *testing.T) {
	testErr := errors.New("unit-test")
	example := map[string][]*types.StepData{
		"Image_Pull": {
			{
				RetryID:  0,
				StepName: "Image_Pull",
				Status:   "Error",
				ErrorMsg: testErr,
			},
			{
				RetryID:  1,
				StepName: "Image_Pull",
				Status:   "Success",
			},
		},
		"HealthCheck": {
			{
				RetryID:  0,
				StepName: "HealthCheck",
				Status:   "Success",
			},
		},
	}
	StartStep(StepMetrics, "Image_Pull")
	EndStep(StepMetrics, "Image_Pull", nil, testErr)

	StartStep(StepMetrics, "Image_Pull")
	EndStep(StepMetrics, "Image_Pull", nil, nil)

	StartStep(StepMetrics, "HealthCheck")
	EndStep(StepMetrics, "HealthCheck", nil, nil)

	for k, v1 := range example {
		v2, ok := StepMetrics[k]
		assert.True(t, ok)
		assert.Equal(t, len(v1), len(v2))
		for i, s1 := range v1 {
			s2 := v2[i]
			assert.Equal(t, (s2.EndTime-s2.StartTime)*1000, s2.ExecTimeMS)
			assert.Equal(t, s1.RetryID, s2.RetryID)
			assert.Equal(t, s1.Status, s2.Status)
			assert.Equal(t, s1.StepName, s2.StepName)
			assert.Equal(t, s1.ErrorMsg, s2.ErrorMsg)
			assert.Equal(t, s1.Tags, s2.Tags)
		}
	}
}

func TestAddSvcContainers(t *testing.T) {
	t.Run("base container list is empty", func(t *testing.T) {
		var to []types.SvcContainer
		res := AddSvcContainers(to, []types.SvcContainer{{ServiceName: "test1"}})
		assert.Equal(t, 1, len(res))
		fmt.Println(res)
	})
	t.Run("add duplicates to base container list", func(t *testing.T) {
		var to []types.SvcContainer
		to = []types.SvcContainer{
			{
				ServiceName: "test1",
			},
			{
				ServiceName: "test2",
			},
		}
		res := AddSvcContainers(to, []types.SvcContainer{{ServiceName: "test2"}})
		assert.Equal(t, 2, len(res))
		assert.Equal(t, "test1", res[0].ServiceName)
		assert.Equal(t, "test2", res[1].ServiceName)
		fmt.Println(res)
	})
	t.Run("add non-duplicates to base container list", func(t *testing.T) {
		var to []types.SvcContainer
		to = []types.SvcContainer{
			{
				ServiceName: "test1",
			},
			{
				ServiceName: "test2",
			},
		}
		res := AddSvcContainers(to, []types.SvcContainer{{ServiceName: "test3"}})
		assert.Equal(t, 3, len(res))
		assert.Equal(t, "test1", res[0].ServiceName)
		assert.Equal(t, "test2", res[1].ServiceName)
		assert.Equal(t, "test3", res[2].ServiceName)
		fmt.Println(res)
	})
	t.Run("add duplicates & non-duplicates to base container list", func(t *testing.T) {
		var to []types.SvcContainer
		to = []types.SvcContainer{
			{
				ServiceName: "test1",
			},
			{
				ServiceName: "test2",
			},
		}
		res := AddSvcContainers(to, []types.SvcContainer{{ServiceName: "test2"}, {ServiceName: "test3"}, {ServiceName: "test4"}})
		assert.Equal(t, 4, len(res))
		assert.Equal(t, "test1", res[0].ServiceName)
		assert.Equal(t, "test2", res[1].ServiceName)
		assert.Equal(t, "test3", res[2].ServiceName)
		assert.Equal(t, "test4", res[3].ServiceName)
		fmt.Println(res)
	})
}
