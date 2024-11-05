package trtest

import (
	"context"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/utils/gopool"
	"github.com/meow-pad/persian/utils/timewheel"
)

type TransferRuntime struct {
	Timer        *timewheel.TimeWheel
	GOPool       *gopool.GoroutinePool
	NacosOptions *NacosOptions
	ServiceNames []string
}

func newTransfer(runtime *TransferRuntime, options *TransferOptions) (*Transfer, error) {
	transfer := &Transfer{
		Runtime: runtime,
		Options: options,
	}
	if err := transfer.init(); err != nil {
		return nil, err
	}
	return transfer, nil
}

type TransferOptions struct {
	TSOptionsArr []*TransferServerOptions
}

type Transfer struct {
	Runtime *TransferRuntime
	Options *TransferOptions

	tServers []*TransferServer
}

func (transfer *Transfer) init() error {
	var servers []*TransferServer
	for i := range transfer.Options.TSOptionsArr {
		tsOptions := transfer.Options.TSOptionsArr[i]
		tsOptions.AllTSOptions = transfer.Options.TSOptionsArr
		if tsServer, err := newTransferServer(transfer.Runtime, tsOptions); err != nil {
			return err
		} else {
			servers = append(servers, tsServer)
		}
	}
	transfer.tServers = servers
	return nil
}

func (transfer *Transfer) Start() error {
	for i := range transfer.tServers {
		ts := transfer.tServers[i]
		if err := ts.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (transfer *Transfer) Stop(ctx context.Context) error {
	for _, ts := range transfer.tServers {
		if err := ts.Stop(ctx); err != nil {
			plog.Error("transfer server stop error", pfield.Error(err))
		}
	}
	return nil
}
