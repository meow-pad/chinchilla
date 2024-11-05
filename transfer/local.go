package transfer

import (
	"context"
	"fmt"
	"github.com/meow-pad/chinchilla/handler"
	"github.com/meow-pad/chinchilla/transfer/common"
	"github.com/meow-pad/chinchilla/transfer/service"
	"github.com/meow-pad/persian/errdef"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	netcodec "github.com/meow-pad/persian/frame/pnet/tcp/codec"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"github.com/meow-pad/persian/frame/pnet/utils"
	"sync/atomic"
)

func newLocalServerSession(local *Local, manager *Manager,
	ctxBuilder func(session.Session) (session.Context, error)) (*localServerSession, error) {
	sess := &localServerSession{
		clientSess: newLocalClientSession(local),
		listener:   newLocalListener(manager),
	}
	ctx, err := ctxBuilder(sess)
	if err != nil {
		return nil, err
	}
	_ = sess.BaseSession.Register(ctx)
	return sess, nil
}

type localServerSession struct {
	session.BaseSession

	clientSess *localClientSession
	listener   *localListener
}

func (sess *localServerSession) Register(context session.Context) error {
	return fmt.Errorf("cant register local serverSession context")
}

func (sess *localServerSession) Connection() session.Conn {
	return nil
}

func (sess *localServerSession) Close() error {
	return nil
}

func (sess *localServerSession) IsClosed() bool {
	return false
}

func (sess *localServerSession) SendMessage(message any) {
	sess.listener.handleMessage(sess.clientSess, message)
}

func (sess *localServerSession) SendMessages(messages ...any) {
	for _, msg := range messages {
		sess.listener.handleMessage(sess.clientSess, msg)
	}
}

func newLocalClientSession(local *Local) *localClientSession {
	return &localClientSession{
		local: local,
	}
}

type localClientSession struct {
	session.BaseSession

	local *Local
}

func (sess *localClientSession) Register(context session.Context) error {
	return fmt.Errorf("cant register local clientSession context")
}

func (sess *localClientSession) Connection() session.Conn {
	return nil
}

func (sess *localClientSession) Close() error {
	return nil
}

func (sess *localClientSession) IsClosed() bool {
	return false
}

func (sess *localClientSession) SendMessage(message any) {
	if err := sess.local.SendMessage(message); err != nil {
		plog.Error("local client sess send message error:", pfield.Error(err))
	}
}

func (sess *localClientSession) SendMessages(messages ...any) {
	for _, msg := range messages {
		if err := sess.local.SendMessage(msg); err != nil {
			plog.Error("local client sess send message error:", pfield.Error(err))
		}
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
	manager       *Manager
	info          common.Info
	serverSession session.Session // 这里比较特殊，它是模拟服务接收端的网关会话（非网关内部的服务的会话），这与网关内其他会话概念相反
	serverCodec   netcodec.Codec
	handler       handler.MessageHandler

	stopped atomic.Bool
}

func (local *Local) init(manager *Manager, info common.Info) error {
	if manager == nil {
		return errdef.ErrInvalidParams
	}
	options := manager.transfer.Options
	if options.LocalMessageHandler == nil {
		return errdef.ErrInvalidParams
	}
	if options.LocalServerCodec == nil {
		return fmt.Errorf("less local server codec ")
	}
	if options.LocalContextBuilder == nil {
		return fmt.Errorf("less local serverSession context builder")
	}
	sess, err := newLocalServerSession(local, manager, options.LocalContextBuilder)
	if err != nil {
		return err
	}
	serviceName := info.Service()
	msgHandler := options.LocalMessageHandler[serviceName]
	if msgHandler == nil {
		return fmt.Errorf("less service(%s) message msgHandler", serviceName)
	}
	local.manager = manager
	local.info = info
	local.serverSession = sess
	local.serverCodec = options.LocalServerCodec
	local.handler = msgHandler
	return nil
}

func (local *Local) UpdateInfo(info common.Info) error {
	if info.ServiceName != local.info.ServiceName || info.ServiceId() != local.info.ServiceId() {
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
	return local.handler.HandleMessage(local.serverSession, msg)
}

func (local *Local) TransferMessage(msgBytes []byte) error {
	if local.stopped.Load() {
		return service.ErrStoppedInstance
	}
	if !local.info.Enable {
		return service.ErrDisabledService
	}
	reader := utils.NewBytesReader(msgBytes)
	if msgArr, _, err := local.serverCodec.Decode(reader); err != nil {
		return err
	} else {
		for _, msg := range msgArr {
			if hErr := local.handler.HandleMessage(local.serverSession, msg); hErr != nil {
				return hErr
			}
		} // end of for
	}
	return nil
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
