package _default

import (
	"context"

	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/plugin"
	"github.com/paypal/dce-go/types"
	"github.com/paypal/dce-go/utils"
	"github.com/paypal/dce-go/utils/pod"
	"github.com/paypal/dce-go/utils/wait"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const name = "default"

type monitor struct{}

func init() {
	// Register default monitor plugin
	log.SetOutput(config.CreateFileAppendMode(types.DCE_OUT))
	plugin.Monitors.Register(&monitor{}, name)
	log.Infof("Registered monitor plugin %s", name)
}

func (m *monitor) Start(ctx context.Context) (types.PodStatus, error) {
	logger := log.WithFields(log.Fields{
		"monitor": name,
	})
	logger.Print("enter monitor plugin")
	// Get infra container ID
	var infraContainerId string
	var err error
	if !config.GetConfig().GetBool(types.RM_INFRA_CONTAINER) {
		infraContainerId, err = pod.GetContainerIdByService(pod.ComposeFiles, types.INFRA_CONTAINER)
		if err != nil {
			return types.POD_FAILED, errors.Wrap(err, "fail to get infra container ID")
		}
		logger.Debugf("Infra container ID: %s", infraContainerId)
	}

	run := func() (types.PodStatus, error) {
		for i := 0; i < len(pod.MonitorContainerList); i++ {
			hc, ok := pod.HealthCheckListId[pod.MonitorContainerList[i]]
			healthy, running, exitCode, err := pod.CheckContainer(pod.MonitorContainerList[i], ok && hc)
			if err != nil {
				return types.POD_FAILED, err
			}
			logger.Debugf("container %s has health check, health status: %s, exitCode: %d, err : %v",
				pod.MonitorContainerList[i], healthy.String(), exitCode, err)

			if exitCode != 0 {
				return types.POD_FAILED, nil
			}

			if healthy == types.UNHEALTHY {
				err = pod.PrintInspectDetail(pod.MonitorContainerList[i])
				if err != nil {
					log.Warnf("failed to get container detail: %s ", err)
				}
				return types.POD_FAILED, nil
			}

			if exitCode == 0 && !running {
				logger.Infof("Removed finished(exit with 0) container %s from monitor list",
					pod.MonitorContainerList[i])
				pod.MonitorContainerList = append(pod.MonitorContainerList[:i], pod.MonitorContainerList[i+1:]...)
				i--

			}
		}

		// Send finished to mesos IF no container running or ONLY system proxy is running in the pod
		switch config.IsService() {
		case true:
			if len(pod.MonitorContainerList) == 0 {
				logger.Error("Task is SERVICE. All containers in the pod exit with code 0, sending FAILED")
				return types.POD_FAILED, nil
			}
			if len(pod.MonitorContainerList) == 1 && pod.MonitorContainerList[0] == infraContainerId {
				logger.Error("Task is SERVICE. Only infra container is running in the pod, sending FAILED")
				return types.POD_FAILED, nil
			}
		case false:
			if len(pod.MonitorContainerList) == 0 {
				logger.Info("Task is ADHOC job. All containers in the pod exit with code 0, sending FINISHED")
				return types.POD_FINISHED, nil
			}
			if len(pod.MonitorContainerList) == 1 && pod.MonitorContainerList[0] == infraContainerId {
				logger.Info("Task is ADHOC job. Only infra container is running in the pod, sending FINISHED")
				return types.POD_FINISHED, nil
			}
		}
		return types.POD_EMPTY, nil
	}

	res, err := wait.PollForever(config.GetPollInterval(), nil, func() (string, error) {
		status, err := run()
		if err != nil {
			// Error won't be considered as pod failure unless pod status is failed
			log.Warnf("error from monitor periodical check: %s", err)
		}
		return status.String(), nil
	})

	return utils.ToPodStatus(res), err
}
