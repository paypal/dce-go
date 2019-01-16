package pod

import (
	"sync"

	"github.com/paypal/dce-go/types"
)

type Status struct {
	sync.RWMutex
	Status types.PodStatus
}
