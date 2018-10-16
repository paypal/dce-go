package wait

import "sync"

type LogCommandStatus struct {
	sync.RWMutex
	IsRunning bool
}
