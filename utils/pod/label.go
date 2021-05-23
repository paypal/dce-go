package pod

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/paypal/dce-go/types"
	"github.com/sirupsen/logrus"
)

func GetServiceDetail(taskInfo *mesos.TaskInfo) types.ServiceDetail {
	var sd types.ServiceDetail
	labels := taskInfo.GetLabels().Labels
	for _, l := range labels {
		if *l.Key == types.SERVICE_DETAIL {
			if err := yaml.Unmarshal([]byte(*l.Value), &sd); err != nil {
				logrus.Warnf("fail to unmarshal %s, err %s", *l.Value, err)
			}
			break
		}
	}

	if sd == nil {
		sd = make(map[string]map[string]interface{})
	}
	return sd
}

func UpdateServiceDetail(taskInfo *mesos.TaskInfo, sd types.ServiceDetail) error {
	var err error
	var found bool
	labels := taskInfo.GetLabels().Labels
	for _, l := range labels {
		if *l.Key == types.SERVICE_DETAIL {
			found = true
			*l.Value, err = marshalToString(sd)
			if err != nil {
				return err
			}
		}
	}

	if !found {
		key := types.SERVICE_DETAIL
		val, err := marshalToString(sd)
		if err != nil {
			return err
		}
		taskInfo.Labels.Labels = append(taskInfo.Labels.Labels, &mesos.Label{
			Key:   &key,
			Value: &val,
		})
	}
	return err
}

func marshalToString(sd types.ServiceDetail) (string, error) {
	out, err := yaml.Marshal(sd)
	if err != nil {
		return "", errors.Wrapf(err, "fail to marshal %+v", sd)
	}
	val := string(out)
	return val, nil
}
