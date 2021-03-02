package utils

import (
	"fmt"

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

/*
SetStepData sets the metricsof the DCE step in the map. If the step already exist in the map, it just updates the end time
*/
func SetStepData(stepData map[interface{}]interface{}, startTime, endTime int64, stepName, status string) {

	log.Info("Step Name :" + stepName)
	log.Info("Start Time : ", startTime)
	log.Info("End Time : ", endTime)

	if len(stepName) == 0 {
		log.Error("error while updating step data for Granular Metrics: step name can't be empty for stepData")
	}
	var stepValue map[string]interface{}
	var ok bool

	stepValue, ok = stepData[stepName].(map[string]interface{})
	if !ok {
		stepValue = make(map[string]interface{})
	}

	stepValue["stepName"] = stepName
	if startTime != 0 {
		stepValue["startTime"] = startTime
	}
	if endTime != 0 {
		stepValue["endTime"] = endTime
		log.Infof("Step Value Interface end time : %v, %T", stepValue["endTime"], stepValue["endTime"])
		log.Infof("Step Value Interface start time : %v, %T", stepValue["startTime"], stepValue["startTime"])
		stepValue["execTimeMS"] = (stepValue["endTime"].(int64) - stepValue["startTime"].(int64)) * 1000
	}
	if len(status) > 0 {
		stepValue["status"] = status
	}
	stepData[stepName] = stepValue
}
