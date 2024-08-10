package receiver

import (
	"github.com/meow-pad/chinchilla/receiver/codec"
	tcodec "github.com/meow-pad/chinchilla/transfer/codec"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"github.com/meow-pad/persian/utils/coding"
	"github.com/meow-pad/persian/utils/worker"
	"reflect"
)

func NewListener(server *Receiver) *Listener {
	return &Listener{server: server}
}

type Listener struct {
	server *Receiver
}

func (listener *Listener) OnOpened(sess session.Session) {
	sessCtx := newSessionContext(listener.server, sess)
	if err := sess.Register(sessCtx); err != nil {
		plog.Error("(receiver) register sess context error:", pfield.Error(err))
		if cErr := sess.Close(); cErr != nil {
			plog.Error("close session error:", pfield.Error(cErr))
		}
	} else {
		plog.Debug("(receiver) open session", pfield.Uint64("sessionId", sess.Id()))
		listener.server.Transfer.Forward(int64(sess.Id()), func(local *worker.GoroutineLocal) {
			local.Set(sess.Id(), sess)
		})
	}
}

func (listener *Listener) OnClosed(sess session.Session) {
	plog.Info("(receiver) close session", pfield.Uint64("sessionId", sess.Id()))
	listener.server.Transfer.Forward(int64(sess.Id()), func(local *worker.GoroutineLocal) {
		// 移除缓存
		local.Remove(sess.Id())
	})
}

func (listener *Listener) OnReceive(sess session.Session, msg any, msgLen int) (err error) {
	if plog.LoggerLevel() == plog.DebugLevel {
		plog.Debug("(receiver) receive message:", pfield.String("msgType", reflect.TypeOf(msg).String()), pfield.JsonString("msg", msg))
	}
	listener.handleMessage(sess, msg)
	return nil
}

func (listener *Listener) OnReceiveMulti(sess session.Session, msgArr []any, totalLen int) (err error) {
	for _, msg := range msgArr {
		if plog.LoggerLevel() == plog.DebugLevel {
			plog.Debug("(receiver) receive message:", pfield.String("msgType", reflect.TypeOf(msg).String()), pfield.JsonString("msg", msg))
		}
		listener.handleMessage(sess, msg)
	}
	return
}

func (listener *Listener) OnSend(sess session.Session, msg any, msgLen int) (err error) {
	if plog.LoggerLevel() == plog.DebugLevel {
		plog.Debug("(receiver) send message:", pfield.String("msgType", reflect.TypeOf(msg).String()), pfield.JsonString("msg", msg))
	}
	return
}

func (listener *Listener) OnSendMulti(sess session.Session, msg []any, totalLen int) (err error) {
	return
}

func (listener *Listener) handleMessage(sess session.Session, msg any) {
	switch req := msg.(type) {
	case *codec.MessageReq:
		listener.handleMessageReq(sess, req)
	case *codec.HeartbeatReq:
		listener.handleHeartbeatReq(sess, req)
	case *codec.HandshakeReq:
		listener.handleHandshakeReq(sess, req)
	}
}

func (listener *Listener) handleMessageReq(sess session.Session, req *codec.MessageReq) {
	listener.server.Transfer.Forward(int64(sess.Id()), func(local *worker.GoroutineLocal) {
		sessCtx := coding.Cast[*SenderContext](sess.Context())
		if sessCtx == nil {
			if cErr := sess.Close(); cErr != nil {
				plog.Error("close session error:", pfield.Error(cErr))
			}
			return
		}
		reqService := req.Service
		if len(reqService) <= 0 {
			reqService, _ = sessCtx.GetDefaultService()
		}
		srvService := sessCtx.GetService(reqService)
		if srvService == nil {
			res := &codec.MessageRes{}
			res.Code = codec.ErrCodeHandshakeFirst
			sess.SendMessage(res)
			return
		}
		if srvService.IsStopped() {
			// 服务停止了？
			if cErr := sess.Close(); cErr != nil {
				plog.Error("(receiver) close session error:", pfield.Error(cErr))
			}
			return
		}
		if sessCtx.IsRegistered() {
			plog.Debug("(receiver) send MessageSReq to service",
				pfield.Uint64("sessId", sessCtx.Id()), pfield.String("service", reqService))
			if err := srvService.SendMessage(&tcodec.MessageSReq{
				ConnId:  sess.Id(),
				Payload: req.Payload,
			}); err != nil {
				plog.Error("(receiver) send message to service error:", pfield.Error(err))
			}
		} else {
			// 先对会话进行注册
			plog.Debug("(receiver) register to service",
				pfield.Uint64("sessId", sessCtx.Id()), pfield.String("service", reqService))
			if err := srvService.SendMessage(&tcodec.RegisterSReq{
				ConnId:  sess.Id(),
				Payload: req.Payload,
			}); err != nil {
				plog.Error("(receiver) send message to service error:", pfield.Error(err))
			}
			return
		}
	})
}

func (listener *Listener) handleHeartbeatReq(sess session.Session, req *codec.HeartbeatReq) {
	listener.server.Transfer.Forward(int64(sess.Id()), func(local *worker.GoroutineLocal) {
		sessCtx := coding.Cast[*SenderContext](sess.Context())
		if sessCtx == nil {
			if cErr := sess.Close(); cErr != nil {
				plog.Error("(receiver) close session error:", pfield.Error(cErr))
			}
			return
		}
		if sessCtx.IsRegistered() {
			_, dfService := sessCtx.GetDefaultService()
			if dfService == nil {
				res := &codec.HeartbeatRes{}
				res.Code = codec.ErrCodeHandshakeFirst
				sess.SendMessage(res)
				return
			}
			if dfService.IsStopped() {
				// 服务停止了？
				if cErr := sess.Close(); cErr != nil {
					plog.Error("(receiver) close session error:", pfield.Error(cErr))
				}
				return
			}
			// 更新过期时间
			sessCtx.UpdateDeadline()
			// 转发消息
			plog.Debug("(receiver) handle HeartbeatSReq", pfield.Uint64("sessId", sessCtx.Id()))
			if err := dfService.SendMessage(&tcodec.HeartbeatSReq{
				ConnId:  sess.Id(),
				Payload: req.Payload,
			}); err != nil {
				plog.Error("(receiver) send message to service error:", pfield.Error(err))
			}
			//// 这里尝试先发
			//res := &codec.HeartbeatRes{}
			//sess.SendMessage(res)
			return
		} else {
			res := &codec.HeartbeatRes{}
			res.Code = codec.ErrCodeLoginFirst
			sess.SendMessage(res)
			return
		}
	})
}

func (listener *Listener) handleHandshakeReq(sess session.Session, req *codec.HandshakeReq) {
	listener.server.Transfer.Forward(int64(sess.Id()), func(local *worker.GoroutineLocal) {
		options := listener.server.Options
		//err := req.InitRouterId()
		//if err != nil {
		//	res := &codec.HandshakeRes{}
		//	res.Code = codec.ErrCodeInvalidRouterId
		//	sess.SendMessage(res)
		//	return
		//}
		// 校验
		if req.AuthKey != options.ReceiverHandshakeAuthKey {
			res := &codec.HandshakeRes{}
			res.Code = codec.ErrCodeInvalidAuthKey
			sess.SendMessage(res)
			return
		}
		sessCtx := coding.Cast[*SenderContext](sess.Context())
		if sessCtx == nil {
			if cErr := sess.Close(); cErr != nil {
				plog.Error("(receiver) close session error:", pfield.Error(cErr))
			}
			return
		}
		// 该服务是否已关注成功
		srvCli := sessCtx.GetService(req.Service)
		if srvCli != nil {
			// 又重握手了一遍
			res := &codec.HandshakeRes{}
			res.Code = codec.ErrCodeSuccess
			sess.SendMessage(res)
			return
		}
		// 设定需要先到 本地缓存或分布式缓存查询, 有则直接在transfer中找到对应的instanceId的client
		// 缓存中没有则到transfer去select一个client
		manager := listener.server.Transfer.GetServiceManager(req.Service)
		if manager == nil {
			res := &codec.HandshakeRes{}
			res.Code = codec.ErrCodeUnknownService
			sess.SendMessage(res)
			return
		} else {
			sErr := listener.server.Transfer.GoPool.Submit(func() {
				srv, sErr := manager.SelectInstance(req.RouterId())
				if sErr != nil {
					res := &codec.HandshakeRes{}
					res.Code = codec.ErrCodeSelectError
					sess.SendMessage(res)
					return
				}
				if srv == nil {
					res := &codec.HandshakeRes{}
					res.Code = codec.ErrCodeLessInstance
					sess.SendMessage(res)
					return
				}
				sessCtx.SetService(req.Service, srv)
				res := &codec.HandshakeRes{}
				res.Code = codec.ErrCodeSuccess
				sess.SendMessage(res)
			})
			if sErr != nil {
				plog.Error("(receiver) submit SelectInstance task in HandshakeReq error:", pfield.Error(sErr))
			}
			return
		}
	})
}
