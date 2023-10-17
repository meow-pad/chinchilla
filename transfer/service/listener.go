package service

import (
	rcodec "github.com/meow-pad/chinchilla/receiver/codec"
	"github.com/meow-pad/chinchilla/receiver/server"
	tcodec "github.com/meow-pad/chinchilla/transfer/codec"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"github.com/meow-pad/persian/utils/worker"
)

type Listener struct {
	client *Remote
}

func (listener *Listener) OnOpened(session session.Session) {
	listener.client.handshake()
}

func (listener *Listener) OnClosed(session session.Session) {
	// 尝试重连
	err := listener.client.connect()
	if err != nil {
		plog.Error("reconnect error:", pfield.Error(err))
	}
}

func (listener *Listener) OnReceive(session session.Session, msg any, msgLen int) (err error) {
	listener.handleMessage(msg)
	return
}

func (listener *Listener) OnReceiveMulti(session session.Session, msgArr []any, totalLen int) (err error) {
	for _, msg := range msgArr {
		listener.handleMessage(msg)
	}
	return
}

func (listener *Listener) OnSend(session session.Session, msg any, msgLen int) (err error) {
	return nil
}

func (listener *Listener) OnSendMulti(session session.Session, msg []any, totalLen int) (err error) {
	return nil
}

func getSessionFromGoLocal(local *worker.GoroutineLocal, connId uint64) session.Session {
	value, ok := local.Get(connId)
	if !ok {
		plog.Debug("lost connection", pfield.Uint64("conn", connId))
		return nil
	}
	sess := value.(session.Session)
	if sess == nil {
		plog.Error("invalid session value", pfield.Uint64("conn", connId))
		return nil
	}
	return sess
}

func (listener *Listener) handleMessage(msg any) {
	switch res := msg.(type) {
	case *tcodec.MessageSRes:
		listener.handleMessageRes(res)
	case *tcodec.RegisterSRes:
		listener.handleRegisterRes(res)
	case *tcodec.UnregisterSRes:
		listener.handleUnregisterRes(res)
	case *tcodec.HeartbeatSRes:
		listener.handleHeartbeatRes(res)
	case *tcodec.HandshakeRes:
		listener.handleHandshakeRes(res)
	}
}

func (listener *Listener) handleMessageRes(res *tcodec.MessageSRes) {
	listener.client.manager.transfer.Forward(int64(res.ConnId), func(local *worker.GoroutineLocal) {
		sess := getSessionFromGoLocal(local, res.ConnId)
		if sess == nil {
			return
		}
		sess.SendMessage(&rcodec.MessageRes{Payload: res.Payload})
	})
}

func (listener *Listener) handleRegisterRes(res *tcodec.RegisterSRes) {
	listener.client.manager.transfer.Forward(int64(res.ConnId), func(local *worker.GoroutineLocal) {
		sess := getSessionFromGoLocal(local, res.ConnId)
		if sess == nil {
			return
		}
		if res.Code == tcodec.ErrCodeSuccess {
			ctx := sess.Context()
			if ctx == nil {
				plog.Error("nil session context")
			} else {
				senderCtx := ctx.(*server.SenderContext)
				if senderCtx == nil {
					plog.Error("invalid session context")
				} else {
					senderCtx.SetRegistered(true)
				}
			} // end of else
		} // end of if
		sess.SendMessage(&rcodec.MessageRes{Payload: res.Payload})
	})
}

func (listener *Listener) handleUnregisterRes(res *tcodec.UnregisterSRes) {
	listener.client.manager.transfer.Forward(int64(res.ConnId), func(local *worker.GoroutineLocal) {
		sess := getSessionFromGoLocal(local, res.ConnId)
		if sess == nil {
			return
		}
		// 关闭连接
		if err := sess.Close(); err != nil {
			plog.Error("close session error:", pfield.Error(err))
		} else {
			local.Remove(res.ConnId)
		}
	})
}

func (listener *Listener) handleHeartbeatRes(res *tcodec.HeartbeatSRes) {
	// nothing
}

func (listener *Listener) handleHandshakeRes(res *tcodec.HandshakeRes) {
	switch res.Code {
	case tcodec.ErrCodeSuccess:
		// 握手成功
		listener.client.onHandshake()
	default:
		plog.Error("handshake error:", pfield.Uint16("code", res.Code))
		// 无法处理的情况则停掉客户端
		if err := listener.client.closeConn(); err != nil {
			plog.Error("close conn error:", pfield.Error(err))
		}
		//go func() {
		//	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		//	defer cancel()
		//	if err := listener.client.Stop(ctx); err != nil {
		//		plog.Error("stop client error:", pfield.Error(err))
		//	}
		//}()
	}
}
