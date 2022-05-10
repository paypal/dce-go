package pod

import (
	"fmt"
	"strings"
	"time"

	"github.com/paypal/dce-go/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type ConditionFunc func() (string, error)

func PluginPanicHandler(condition ConditionFunc) (res string, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recover : %v \n", r)
			err = errors.New(fmt.Sprintf("Recover : %v \n", r))
		}
	}()

	if res, err = condition(); err != nil {
		log.Errorf("Error executing plugins: %v \n", err)
		return res, err
	}
	return res, err
}

func ToPodStatus(s string) types.PodStatus {
	switch s {
	case "POD_STAGING":
		return types.POD_STAGING
	case "POD_STARTING":
		return types.POD_STARTING
	case "POD_RUNNING":
		return types.POD_RUNNING
	case "POD_FAILED":
		return types.POD_FAILED
	case "POD_KILLED":
		return types.POD_KILLED
	case "POD_FINISHED":
		return types.POD_FINISHED
	case "POD_PULL_FAILED":
		return types.POD_PULL_FAILED
	case "POD_COMPOSE_CHECK_FAILED":
		return types.POD_COMPOSE_CHECK_FAILED
	}

	return types.POD_EMPTY
}

func ToHealthStatus(s string) types.HealthStatus {
	switch s {
	case "starting":
		return types.STARTING
	case "healthy":
		return types.HEALTHY
	case "unhealthy":
		return types.UNHEALTHY
	}

	return types.UNKNOWN_HEALTH_STATUS
}

// StartStep start a step of dce, and add a new item into the values, key is the stepName
// In the map, key will be step, value will be each retry result, and duration.
func StartStep(stepData map[string][]*types.StepData, stepName string) {
	if len(stepName) == 0 {
		log.Error("error while updating step data for Granular Metrics: step name can't be empty for stepData")
	}
	var ok bool

	stepValues, ok := stepData[stepName]
	if !ok {
		stepValues = []*types.StepData{}
		stepData[stepName] = stepValues
	}
	stepValue := &types.StepData{}

	stepValue.StepName = stepName
	stepValue.RetryID = len(stepValues)
	stepValue.StartTime = time.Now().Unix()
	stepValue.Status = "Starting"
	stepValues = append(stepValues, stepValue)
	stepData[stepName] = stepValues
}

// EndStep ends the current dce step, and update the result, duraiton.
// current dce step can be fetch from stepData, key is the stepName, value is each retry results. Update the latest result
func EndStep(stepData map[string][]*types.StepData, stepName string, tag map[string]string, err error) {
	if len(stepName) == 0 {
		log.Error("error while updating step data for Granular Metrics: step name can't be empty for stepData")
		return
	}
	var ok bool

	stepValues, ok := stepData[stepName]
	if !ok {
		log.Errorf("key %s not exist in stepData %+v", stepName, stepData)
		return
	}
	if len(stepValues) < 1 {
		log.Errorf("len of stepValues is %d, less than 1", len(stepValues))
		return
	}

	step := stepValues[len(stepValues)-1]
	step.Tags = tag
	step.EndTime = time.Now().Unix()
	step.ErrorMsg = err
	step.ExecTimeMS = (step.EndTime - step.StartTime) * 1000
	if err != nil {
		step.Status = "Error"
	} else if healthStatus, ok := tag["healthStatus"]; ok && healthStatus != "healthy" {
		step.Status = "Error"
	} else {
		step.Status = "Success"
	}
}

func UpdateHealthCheckStatus(stepData map[string][]*types.StepData) {
	for stepName, stepVals := range stepData {
		if strings.HasPrefix(stepName, "HealthCheck-") &&
			len(stepVals) > 0 &&
			stepVals[len(stepVals)-1].Status == "Starting" {

			stepVals[len(stepVals)-1].Status = "Error"
		}
	}
}
