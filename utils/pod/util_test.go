package pod

import (
	"errors"
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
