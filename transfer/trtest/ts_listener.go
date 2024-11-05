package trtest

import (
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"reflect"
)

const (
	withoutHandshakeTransferTimeoutMs = 60 * 1000
)

type TSListener struct {
	msgHandler *TSHandler
}

func (listener *TSListener) OnOpened(sess session.Session) {
	plog.Info("==transfer server session opened:",
		pfield.String("serviceId", listener.msgHandler.Server.ServiceId()),
		pfield.String("remoteAddr", sess.Connection().RemoteAddr().String()))
	sessCtx, err := newRemoteContext()
	if err == nil {
		err = sess.Register(sessCtx)
	}
	if err != nil {
		plog.Error("register transfer session error:", pfield.Error(err))
		if err = sess.Close(); err != nil {
			plog.Error("close transfer session error:", pfield.Error(err))
		}
	}
}

func (listener *TSListener) OnClosed(sess session.Session) {
	plog.Info("transfer server session closed:",
		pfield.String("remoteAddr", sess.Connection().RemoteAddr().String()))
}

func (listener *TSListener) OnReceive(sess session.Session, msg any, msgLen int) error {
	plog.Debug("transfer server receive message:",
		pfield.String("", reflect.TypeOf(msg).String()),
		pfield.JsonString("msg", msg),
	)
	if err := listener.msgHandler.HandleMessage(sess, msg); err != nil {
		plog.Error("handle message error:", pfield.Error(err))
	}
	return nil
}

func (listener *TSListener) OnReceiveMulti(sess session.Session, msgArr []any, totalLen int) error {
	for _, msg := range msgArr {
		if plog.LoggerLevel() == plog.DebugLevel {
			plog.Debug("(monopoly-go) receive message on multi:",
				pfield.String("", reflect.TypeOf(msg).String()),
				pfield.JsonString("msg", msg),
			)
		}
		if err := listener.msgHandler.HandleMessage(sess, msg); err != nil {
			plog.Error("handle message error:", pfield.Error(err))
		}
	}
	return nil
}

func (listener *TSListener) OnSend(sess session.Session, msg any, msgLen int) (err error) {
	plog.Debug("(monopoly-go) send message:", pfield.JsonString("msg", msg))
	return
}

func (listener *TSListener) OnSendMulti(sess session.Session, msg []any, totalLen int) (err error) {
	return
}
