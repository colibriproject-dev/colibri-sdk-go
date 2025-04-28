package observer

import (
	"context"
	"sync"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

var once sync.Once
var singleInstance *sync.WaitGroup

func GetWaitGroup() *sync.WaitGroup {
	if singleInstance == nil {
		once.Do(func() {
			logging.Debug(context.Background()).Msg("Creating single WaitGroup instance now.")
			singleInstance = &sync.WaitGroup{}
		})
	} else {
		logging.Debug(context.Background()).Msg("Single WaitGroup instance already created.")
	}

	return singleInstance
}

func WaitRunningTimeout() bool {
	timeout := config.WAIT_GROUP_TIMEOUT_SECONDS
	c := make(chan struct{})

	go func() {
		defer close(c)
		GetWaitGroup().Wait()
	}()

	select {
	case <-c:
		return false
	case <-time.After(time.Duration(timeout) * time.Second):
		return true
	}
}
