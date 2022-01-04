package ocpp

import (
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

const timeout = time.Minute

type CP struct {
	mu          sync.Mutex
	log         *util.Logger
	id          string
	txnCount    int64
	meterValues []types.MeterValue

	bootWG sync.WaitGroup
	boot   *core.BootNotificationRequest
	status *core.StatusNotificationRequest
}

// Boot waits for the CP to register itself
func (cp *CP) Boot() error {
	cp.bootWG.Add(2) // boot and status

	bootC := make(chan struct{})
	go func() {
		cp.bootWG.Wait()
		close(bootC)
	}()

	select {
	case <-bootC:
		return nil
	case <-time.After(timeout):
		return api.ErrTimeout
	}
}

func (cp *CP) Status() (api.ChargeStatus, error) {
	// timeoutTimestamp := time.Now().Add(timeout)

	// for cp.status.Timestamp != nil || time.Now().Before(timeoutTimestamp) {
	// 	cp.log.TRACE.Printf("waiting for status from charge point %s", cp.id)
	// 	time.Sleep(5 * time.Second)

	// 	if cp.status.Timestamp != nil {
	// 		break
	// 	}
	// }

	cp.mu.Lock()
	defer cp.mu.Unlock()

	res := api.StatusNone

	if time.Since(cp.status.Timestamp.Time) > timeout {
		return res, api.ErrTimeout
	}

	if cp.status.ErrorCode != core.NoError {
		return res, fmt.Errorf("chargepoint error: %s", cp.status.ErrorCode)
	}

	switch cp.status.Status {
	case core.ChargePointStatusUnavailable: // "Unavailable"
		res = api.StatusA
	case core.ChargePointStatusAvailable, // "Available"
		core.ChargePointStatusPreparing,     // "Preparing"
		core.ChargePointStatusSuspendedEVSE, // "SuspendedEVSE"
		core.ChargePointStatusSuspendedEV,   // "SuspendedEV"
		core.ChargePointStatusFinishing:     // "Finishing"
		res = api.StatusB
	case core.ChargePointStatusCharging: // "Charging"
		res = api.StatusC
	case core.ChargePointStatusReserved, // "Reserved"
		core.ChargePointStatusFaulted: // "Faulted"
		return api.StatusF, fmt.Errorf("chargepoint status: %s", cp.status.Status)
	default:
		return api.StatusNone, fmt.Errorf("invalid chargepoint status: %s", cp.status.Status)
	}

	return res, nil
}
