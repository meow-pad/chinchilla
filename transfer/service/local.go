package service

import (
	"context"
	"fmt"
	"github.com/meow-pad/chinchilla/handler"
	"github.com/meow-pad/persian/errdef"
	"sync/atomic"
)

func newLocalService(manager *Manager, info Info) (*Local, error) {
	local := &Local{}
	if err := local.init(manager, info); err != nil {
		return nil, err
	}
	return local, nil
}

type Local struct {
	manager *Manager
	info    Info
	handler handler.MessageHandler

	stopped atomic.Bool
}

func (local *Local) init(manager *Manager, info Info) error {
	options := manager.transfer.Options
	if manager == nil || options.ServiceMessageHandler == nil {
		return errdef.ErrInvalidParams
	}
	msgHandler := options.ServiceMessageHandler[info.ServiceName]
	if msgHandler == nil {
		return fmt.Errorf("less service(%s) message msgHandler", info.ServiceName)
	}
	local.manager = manager
	local.info = info
	local.handler = msgHandler
	return nil
}

func (local *Local) UpdateInfo(info Info) error {
	if info.ServiceName != local.info.ServiceName || info.InstanceId != local.info.InstanceId {
		return errdef.ErrInvalidParams
	}
	local.info = info
	return nil
}

func (local *Local) Info() Info {
	return local.info
}

func (local *Local) KeepAlive() bool {
	return true
}

func (local *Local) SendMessage(msg any) error {
	if local.stopped.Load() {
		return ErrStoppedInstance
	}
	if !local.info.Enable {
		return ErrDisabledService
	}
	return local.handler.HandleMessage(msg)
}

func (local *Local) IsEnable() bool {
	if local.stopped.Load() {
		return false
	}
	return local.info.Enable
}

func (local *Local) Stop(ctx context.Context) error {
	local.stopped.Store(true)
	return nil
}

func (local *Local) IsStopped() bool {
	return local.stopped.Load()
}
