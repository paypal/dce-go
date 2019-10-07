package pod

import (
	"sync"

	"github.com/paypal/dce-go/types"
)

type Status struct {
	sync.RWMutex
	Status      types.PodStatus
	// if set to true, indicates that the pod was launched successfully and task moved to RUNNING state
	Launched bool
}
