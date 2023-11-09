package transfer

import (
	"context"
	"fmt"
	"github.com/meow-pad/chinchilla/handler"
	"github.com/meow-pad/chinchilla/transfer/common"
	"github.com/meow-pad/chinchilla/transfer/service"
	"github.com/meow-pad/persian/errdef"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"sync/atomic"
)

func newLocalSession(manager *Manager,
	ctxBuilder func(session.Session) (session.Context, error)) (*localSession, error) {
	sess := &localSession{
		listener: newLocalListener(manager),
	}
	ctx, err := ctxBuilder(sess)
	if err != nil {
		return nil, err
	}
	sess.ctx = ctx
	return sess, nil
}

type localSession struct {
	session.BaseSession

	ctx      session.Context
	listener *localListener
}

func (sess *localSession) Register(context session.Context) error {
	return fmt.Errorf("cant register local session context")
}

func (sess *localSession) Connection() session.Conn {
	return nil
}

func (sess *localSession) Close() error {
	return nil
}

func (sess *localSession) IsClosed() bool {
	return false
}

func (sess *localSession) SendMessage(message any) {
	sess.listener.handleMessage(message)
}

func (sess *localSession) SendMessages(messages ...any) {
	for _, msg := range messages {
		sess.listener.handleMessage(msg)
	}
}

// NewLocalService
//
//	@Description: 构建本地服务
//	@param manager
//	@param info
//	@return *Local
//	@return error
func NewLocalService(manager *Manager, info common.Info) (*Local, error) {
	local := &Local{}
	if err := local.init(manager, info); err != nil {
		return nil, err
	}
	return local, nil
}

type Local struct {
	manager *Manager
	info    common.Info
	session session.Session
	handler handler.MessageHandler

	stopped atomic.Bool
}

func (local *Local) init(manager *Manager, info common.Info) error {
	options := manager.transfer.Options
	if manager == nil || options.LocalMessageHandler == nil {
		return errdef.ErrInvalidParams
	}
	if options.LocalContextBuilder == nil {
		return fmt.Errorf("less local session context builder")
	}
	sess, err := newLocalSession(local.manager, options.LocalContextBuilder)
	if err != nil {
		return err
	}
	msgHandler := options.LocalMessageHandler[info.ServiceName]
	if msgHandler == nil {
		return fmt.Errorf("less service(%s) message msgHandler", info.ServiceName)
	}
	local.manager = manager
	local.info = info
	local.session = sess
	local.handler = msgHandler
	return nil
}

func (local *Local) UpdateInfo(info common.Info) error {
	if info.ServiceName != local.info.ServiceName || info.InstanceId != local.info.InstanceId {
		return errdef.ErrInvalidParams
	}
	local.info = info
	return nil
}

func (local *Local) Info() common.Info {
	return local.info
}

func (local *Local) KeepAlive() bool {
	return true
}

func (local *Local) SendMessage(msg any) error {
	if local.stopped.Load() {
		return service.ErrStoppedInstance
	}
	if !local.info.Enable {
		return service.ErrDisabledService
	}
	return local.handler.HandleMessage(local.session, msg)
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
