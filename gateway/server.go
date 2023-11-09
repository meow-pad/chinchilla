package gateway

import (
	"context"
	"github.com/meow-pad/chinchilla/option"
	"github.com/meow-pad/chinchilla/receiver"
	"github.com/meow-pad/chinchilla/transfer"
	"github.com/meow-pad/persian/frame/pboot"
	"github.com/meow-pad/persian/frame/pservice/cache"
	"github.com/meow-pad/persian/utils/timewheel"
	"github.com/pkg/errors"
)

func NewGateway(
	appInfo pboot.AppInfo,
	secTimer *timewheel.TimeWheel,
	cache *cache.Cache,
	options *option.Options,
) (*Gateway, error) {
	gw := &Gateway{
		appInfo:  appInfo,
		secTimer: secTimer,
		cache:    cache,
		options:  options,
	}
	if gw.appInfo == nil {
		return nil, errors.WithStack(errors.New("nil appInfo"))
	}
	if gw.secTimer == nil {
		return nil, errors.WithStack(errors.New("nil secTimer"))
	}
	if gw.cache == nil {
		return nil, errors.WithStack(errors.New("nil cache"))
	}
	if gw.options == nil {
		return nil, errors.WithStack(errors.New("nil options"))
	}
	err := gw.init()
	if err != nil {
		return nil, err
	}
	return gw, nil
}

type Gateway struct {
	appInfo  pboot.AppInfo
	secTimer *timewheel.TimeWheel
	cache    *cache.Cache
	options  *option.Options

	transfer *transfer.Transfer
	receiver *receiver.Receiver
}

func (gw *Gateway) init() error {
	var err error
	gw.transfer, err = transfer.NewTransfer(gw.appInfo, gw.secTimer, gw.cache, gw.options)
	if err != nil {
		return err
	}
	gw.receiver, err = receiver.NewReceiver(gw.transfer, gw.options)
	if err != nil {
		return err
	}
	return nil
}

func (gw *Gateway) Start(ctx context.Context) error {
	if err := gw.transfer.Start(ctx); err != nil {
		return err
	}
	if err := gw.receiver.Start(ctx); err != nil {
		return err
	}
	return nil
}

func (gw *Gateway) Stop(ctx context.Context) error {
	if err := gw.receiver.Stop(ctx); err != nil {
		return err
	}
	if err := gw.transfer.Stop(ctx); err != nil {
		return err
	}
	return nil
}
