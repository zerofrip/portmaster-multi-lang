package wireguard

import (
	"sync"
	"time"

	"github.com/safing/portmaster/base/database/record"
	"github.com/safing/portmaster/base/runtime"
)

type Status struct {
	record.Base
	sync.Mutex

	State          string
	LastError      string
	ConnectedSince *time.Time
}

var (
	currentStatus = &Status{State: "disabled"}
	pushStatus    runtime.PushFunc
)

func registerStatus() (err error) {
	currentStatus.SetKey("runtime:wireguard/status")
	currentStatus.UpdateMeta()
	pushStatus, err = runtime.Register("wireguard/status", runtime.ProvideRecord(currentStatus))
	return err
}

func setStatus(state, lastError string, connectedSince *time.Time) {
	currentStatus.Lock()
	defer currentStatus.Unlock()
	currentStatus.State = state
	currentStatus.LastError = lastError
	currentStatus.ConnectedSince = connectedSince
	currentStatus.UpdateMeta()
	if pushStatus != nil {
		pushStatus(currentStatus)
	}
}
