package trtest

import (
	"context"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/utils/collections"
	"github.com/meow-pad/persian/utils/gopool"
	"github.com/meow-pad/persian/utils/timewheel"
	"time"
)

type TransferTestOptions struct {
	TransferOptions TransferOptions
	GatewayOptions  GatewayOptions
	NacosOptions    NacosOptions
}

func NewTransferTest(options *TransferTestOptions) (*TransferTest, error) {
	test := &TransferTest{
		Options: options,
	}
	if err := test.init(); err != nil {
		return nil, err
	}
	return test, nil
}

type TransferTest struct {
	Options *TransferTestOptions

	runtime  *TransferRuntime
	timer    *timewheel.TimeWheel
	goPool   *gopool.GoroutinePool
	naming   *NacosNaming
	transfer *Transfer
	gateway  *Gateway
}

func (test *TransferTest) init() error {
	goPool, gErr := gopool.NewGoroutinePool("goPool", 100, 1000, true)
	if gErr != nil {
		return gErr
	} else {
		test.goPool = goPool
	}
	if tw, err := timewheel.NewTimeWheel(1*time.Second, 4, timewheel.WithGoPool(test.goPool)); err != nil {
		return err
	} else {
		test.timer = tw
	}
	var serviceNames []string
	for _, tsOpt := range test.Options.TransferOptions.TSOptionsArr {
		if !collections.IsInSlice[string](serviceNames, tsOpt.ServiceName) {
			serviceNames = append(serviceNames, tsOpt.ServiceName)
		}
	}
	test.runtime = &TransferRuntime{
		Timer:        test.timer,
		ServiceNames: serviceNames,
		NacosOptions: &test.Options.NacosOptions,
		GOPool:       goPool,
	}
	if transfer, err := newTransfer(test.runtime, &test.Options.TransferOptions); err != nil {
		return err
	} else {
		test.transfer = transfer
	}
	if gateway, err := newGateway(test.runtime, &test.Options.GatewayOptions); err != nil {
		return err
	} else {
		test.gateway = gateway
	}
	return nil
}

func (test *TransferTest) Start() error {
	if err := test.goPool.Start(); err != nil {
		return err
	}
	test.timer.Start()
	if err := test.naming.Start(); err != nil {
		return err
	}
	if err := test.transfer.Start(); err != nil {
		return err
	}
	if err := test.gateway.Start(); err != nil {
		return err
	}
	return nil
}

func (test *TransferTest) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	if err := test.gateway.Stop(ctx); err != nil {
		plog.Error("stop gateway error:", pfield.Error(err))
	}
	if err := test.transfer.Stop(ctx); err != nil {
		plog.Error("stop transfer error:", pfield.Error(err))
	}
	if err := test.naming.Stop(); err != nil {
		plog.Error("stop naming error:", pfield.Error(err))
	}
	test.timer.Stop()
	if err := test.goPool.Stop(ctx); err != nil {
		plog.Error("stop goPool error:", pfield.Error(err))
	}
	return nil
}
