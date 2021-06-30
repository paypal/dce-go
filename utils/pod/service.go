package pod

import (
	"github.com/paypal/dce-go/types"
)

func GetServiceDetail() types.ServiceDetail {
	return ServiceDetail
}

func SetServiceDetail(sd types.ServiceDetail) {
	ServiceDetail = sd
}
