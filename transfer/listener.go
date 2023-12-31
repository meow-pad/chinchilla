package transfer

import (
	rcodec "github.com/meow-pad/chinchilla/receiver/codec"
	"github.com/meow-pad/chinchilla/receiver/context"
	tcodec "github.com/meow-pad/chinchilla/transfer/codec"
	"github.com/meow-pad/chinchilla/transfer/common"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"github.com/meow-pad/persian/utils/worker"
	"reflect"
)

type listener struct {
	manager       *Manager
	handleMessage func(session session.Session, msg any)
}

func (listener *listener) OnOpened(session session.Session) {
}

func (listener *listener) OnClosed(session session.Session) {
}

func (listener *listener) OnReceive(session session.Session, msg any, msgLen int) (err error) {
	listener.handleMessage(session, msg)
	return
}

func (listener *listener) OnReceiveMulti(session session.Session, msgArr []any, totalLen int) (err error) {
	for _, msg := range msgArr {
		listener.handleMessage(session, msg)
	}
	return
}

func (listener *listener) OnSend(session session.Session, msg any, msgLen int) (err error) {
	return nil
}

func (listener *listener) OnSendMulti(session session.Session, msg []any, totalLen int) (err error) {
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
		plog.Error("invalid serverSession value", pfield.Uint64("conn", connId))
		return nil
	}
	return sess
}

func (listener *listener) handleMessageRes(res *tcodec.MessageSRes) {
	listener.manager.transfer.Forward(int64(res.ConnId), func(local *worker.GoroutineLocal) {
		sess := getSessionFromGoLocal(local, res.ConnId)
		if sess == nil {
			return
		}
		rRes := &rcodec.MessageRes{}
		rRes.Payload = res.Payload
		sess.SendMessage(rRes)
	})
}

func (listener *listener) handleRegisterRes(res *tcodec.RegisterSRes) {
	listener.manager.transfer.Forward(int64(res.ConnId), func(local *worker.GoroutineLocal) {
		sess := getSessionFromGoLocal(local, res.ConnId)
		if sess == nil {
			return
		}
		if res.Code == common.ErrCodeSuccess {
			ctx := sess.Context()
			if ctx == nil {
				plog.Error("nil serverSession context")
			} else {
				senderCtx := ctx.(context.SenderContext)
				if senderCtx == nil {
					plog.Error("invalid serverSession context")
				} else {
					senderCtx.SetRegistered(true)
				}
			} // end of else
		} // end of if
		rRes := &rcodec.MessageRes{}
		rRes.Payload = res.Payload
		sess.SendMessage(rRes)
	})
}

func (listener *listener) handleUnregisterRes(res *tcodec.UnregisterSRes) {
	listener.manager.transfer.Forward(int64(res.ConnId), func(local *worker.GoroutineLocal) {
		sess := getSessionFromGoLocal(local, res.ConnId)
		if sess == nil {
			return
		}
		// 关闭连接
		if err := sess.Close(); err != nil {
			plog.Error("close serverSession error:", pfield.Error(err))
		} else {
			local.Remove(res.ConnId)
		}
	})
}

func (listener *listener) handleMessageRouter(session session.Session, res *tcodec.MessageRouter) {
	// 注意，这里提交到池子，会导致消息处理顺序变得无序，有要求时需要修改
	err := listener.manager.transfer.GoPool.Submit(func() {
		var err any
		defer func() {
			if err == nil {
				err = recover()
			}
			if err != nil {
				plog.Error("route message error:",
					pfield.Int16("routerType", res.RouterType),
					pfield.String("routerId", res.RouterId),
					pfield.Any("error", err))
			}
		}()
		err = listener.manager.Route(res.RouterType, res.RouterId, res.Payload)
	})
	if err != nil {
		plog.Error("submit router task error:",
			pfield.Int16("routerType", res.RouterType),
			pfield.String("routerId", res.RouterId),
			pfield.Error(err))
	}
}

// newLocalListener
//
//	@Description: 构建 localListener
//	@param manager
//	@return *localListener
func newLocalListener(manager *Manager) *localListener {
	lListener := &localListener{}
	lListener.listener = &listener{
		manager:       manager,
		handleMessage: lListener.handleMessage,
	}
	return lListener
}

type localListener struct {
	*listener
}

func (listener *localListener) handleMessage(session session.Session, msg any) {
	switch res := msg.(type) {
	case *tcodec.MessageSRes:
		listener.handleMessageRes(res)
	case *tcodec.MessageRouter:
		listener.handleMessageRouter(session, res)
	case *tcodec.RegisterSRes:
		listener.handleRegisterRes(res)
	case *tcodec.UnregisterSRes:
		listener.handleUnregisterRes(res)
	case *tcodec.HeartbeatSRes:
	case *tcodec.HandshakeRes:
	case *tcodec.SegmentMsg:
	default:
		plog.Error("unknown message type:", pfield.String("msgType", reflect.TypeOf(msg).String()))
	}
}

// newRemoteListener
//
//	@Description: 构建 remoteListener
//	@param client
//	@return *remoteListener
func newRemoteListener(client *Remote) *remoteListener {
	rListener := &remoteListener{
		client: client,
	}
	rListener.listener = &listener{
		manager:       client.manager,
		handleMessage: rListener.handleMessage,
	}
	return rListener
}

type remoteListener struct {
	*listener

	client *Remote
}

func (listener *remoteListener) OnOpened(session session.Session) {
	listener.client.handshake()
}

func (listener *remoteListener) OnClosed(session session.Session) {
	// 尝试重连
	err := listener.client.Connect()
	if err != nil {
		plog.Error("reconnect error:", pfield.Error(err))
	}
}

func (listener *remoteListener) handleMessage(session session.Session, msg any) {
	switch res := msg.(type) {
	case *tcodec.MessageSRes:
		listener.handleMessageRes(res)
	case *tcodec.MessageRouter:
		listener.handleMessageRouter(session, res)
	case *tcodec.RegisterSRes:
		listener.handleRegisterRes(res)
	case *tcodec.UnregisterSRes:
		listener.handleUnregisterRes(res)
	case *tcodec.HeartbeatSRes:
		listener.handleHeartbeatRes(res)
	case *tcodec.HandshakeRes:
		listener.handleHandshakeRes(res)
	case *tcodec.SegmentMsg:
		listener.handleSegmentMsg(session, res)
	default:
		plog.Error("unknown message type:", pfield.String("msgType", reflect.TypeOf(msg).String()))
	}
}

func (listener *remoteListener) handleHeartbeatRes(res *tcodec.HeartbeatSRes) {
	// nothing
}

func (listener *remoteListener) handleHandshakeRes(res *tcodec.HandshakeRes) {
	switch res.Code {
	case common.ErrCodeSuccess:
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

func (listener *remoteListener) handleSegmentMsg(session session.Session, msg *tcodec.SegmentMsg) {
	if msg.Seq == 0 {
		if listener.client.segmentFrameBuf != nil {
			plog.Error("last segmentation message did not complete")
		}
		listener.client.segmentFrameBuf = msg.Frame
		listener.client.segmentFrameAmount = msg.Amount
		return
	}
	if listener.client.segmentFrameBuf == nil {
		plog.Error("invalid segmentation message buf")
		// 此时无法处理，直接丢弃
		return
	} else if listener.client.segmentFrameAmount != msg.Amount {
		plog.Error("invalid segmentation message amount",
			pfield.Uint16("oldAmount", listener.client.segmentFrameAmount),
			pfield.Uint16("newAmount", msg.Amount))
		// 此时无法处理，直接丢弃
		listener.client.segmentFrameBuf = nil
		listener.client.segmentFrameAmount = 0
		return
	}
	listener.client.segmentFrameBuf = append(listener.client.segmentFrameBuf, msg.Frame...)
	if msg.Seq == msg.Amount-1 {
		buf := listener.client.segmentFrameBuf
		// 清理
		listener.client.segmentFrameBuf = nil
		listener.client.segmentFrameAmount = 0
		// 处理分段消息
		sMsg, err := listener.client.manager.clientCodec.Decode(buf)
		if err != nil {
			plog.Error("decode segmentation message error:", pfield.Error(err))
			return
		}
		listener.handleMessage(session, sMsg)
	}
}
