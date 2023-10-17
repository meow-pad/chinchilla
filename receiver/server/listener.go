package server

import (
	"chinchilla/receiver/codec"
	tcodec "chinchilla/transfer/codec"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"github.com/meow-pad/persian/utils/coding"
	"github.com/meow-pad/persian/utils/worker"
)

type sListener struct {
	server *Server
}

func (listener *sListener) OnOpened(sess session.Session) {
	sessCtx := newSessionContext(listener.server, sess)
	if err := sess.Register(sessCtx); err != nil {
		plog.Error("register sess context error:", pfield.Error(err))
		if cErr := sess.Close(); cErr != nil {
			plog.Error("close session error:", pfield.Error(cErr))
		}
	} else {
		listener.server.Transfer.Forward(int64(sess.Id()), func(local *worker.GoroutineLocal) {
			local.Set(sess.Id(), sess)
		})
	}
}

func (listener *sListener) OnClosed(session session.Session) {
	listener.server.Transfer.Forward(int64(session.Id()), func(local *worker.GoroutineLocal) {
		// 移除缓存
		local.Remove(session.Id())
	})
}

func (listener *sListener) OnReceive(sess session.Session, msg any, msgLen int) (err error) {
	listener.handleMessage(sess, msg)
	return nil
}

func (listener *sListener) OnReceiveMulti(sess session.Session, msgArr []any, totalLen int) (err error) {
	for _, msg := range msgArr {
		listener.handleMessage(sess, msg)
	}
	return
}

func (listener *sListener) OnSend(sess session.Session, msg any, msgLen int) (err error) {
	return
}

func (listener *sListener) OnSendMulti(sess session.Session, msg []any, totalLen int) (err error) {
	return
}

func (listener *sListener) handleMessage(sess session.Session, msg any) {
	switch req := msg.(type) {
	case *codec.MessageReq:
		listener.handleMessageReq(sess, req)
	case *codec.HeartbeatReq:
		listener.handleHeartbeatReq(sess, req)
	case *codec.HandshakeReq:
		listener.handleHandshakeReq(sess, req)
	}
}

func (listener *sListener) handleMessageReq(sess session.Session, req *codec.MessageReq) {
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
			sess.SendMessage(&codec.MessageRes{Code: codec.ErrCodeHandshakeFirst})
			return
		}
		if srvService.IsStopped() {
			// 服务停止了？
			if cErr := sess.Close(); cErr != nil {
				plog.Error("close session error:", pfield.Error(cErr))
			}
			return
		}
		if sessCtx.IsRegistered() {
			if err := srvService.SendMessage(&tcodec.MessageSReq{
				ConnId:  sess.Id(),
				Payload: req.Payload,
			}); err != nil {
				plog.Error("send message to service error:", pfield.Error(err))
			}
		} else {
			if err := srvService.SendMessage(&tcodec.RegisterSReq{
				ConnId:  sess.Id(),
				Payload: req.Payload,
			}); err != nil {
				plog.Error("send message to service error:", pfield.Error(err))
			}
			return
		}
	})
}

func (listener *sListener) handleHeartbeatReq(sess session.Session, req *codec.HeartbeatReq) {
	listener.server.Transfer.Forward(int64(sess.Id()), func(local *worker.GoroutineLocal) {
		sessCtx := coding.Cast[*SenderContext](sess.Context())
		if sessCtx == nil {
			if cErr := sess.Close(); cErr != nil {
				plog.Error("close session error:", pfield.Error(cErr))
			}
			return
		}
		if sessCtx.IsRegistered() {
			_, dfService := sessCtx.GetDefaultService()
			if dfService == nil {
				sess.SendMessage(&codec.HeartbeatRes{Code: codec.ErrCodeHandshakeFirst})
				return
			}
			if dfService.IsStopped() {
				// 服务停止了？
				if cErr := sess.Close(); cErr != nil {
					plog.Error("close session error:", pfield.Error(cErr))
				}
				return
			}
			// 更新过期时间
			sessCtx.UpdateDeadline()
			// 转发消息
			if err := dfService.SendMessage(&tcodec.HeartbeatSReq{
				ConnId:  sess.Id(),
				Payload: req.Payload,
			}); err != nil {
				plog.Error("send message to service error:", pfield.Error(err))
			}
		} else {
			sess.SendMessage(&codec.HeartbeatRes{Code: codec.ErrCodeLoginFirst})
			return
		}
	})
}

func (listener *sListener) handleHandshakeReq(sess session.Session, req *codec.HandshakeReq) {
	listener.server.Transfer.Forward(int64(sess.Id()), func(local *worker.GoroutineLocal) {
		options := listener.server.Gateway.Options
		// 校验
		if req.AuthKey != options.ReceiverAuthKey {
			sess.SendMessage(&codec.HandshakeRes{Code: codec.ErrCodeInvalidAuthKey})
			return
		}
		sessCtx := coding.Cast[*SenderContext](sess.Context())
		if sessCtx == nil {
			if cErr := sess.Close(); cErr != nil {
				plog.Error("close session error:", pfield.Error(cErr))
			}
			return
		}
		// 该服务是否已关注成功
		srvCli := sessCtx.GetService(req.Service)
		if srvCli != nil {
			// 又重握手了一遍
			sess.SendMessage(&codec.HandshakeRes{Code: codec.ErrCodeSuccess})
			return
		}
		// 设定需要先到 本地缓存或分布式缓存查询, 有则直接在transfer中找到对应的instanceId的client
		// 缓存中没有则到transfer去select一个client
		manager := listener.server.Transfer.GetServiceManager(req.Service)
		if manager == nil {
			sess.SendMessage(&codec.HandshakeRes{Code: codec.ErrCodeUnknownService})
			return
		} else {
			srv, err := manager.SelectInstance(req.RouterId)
			if err != nil {
				sess.SendMessage(&codec.HandshakeRes{Code: codec.ErrCodeSelectError})
				return
			}
			if srv == nil {
				sess.SendMessage(&codec.HandshakeRes{Code: codec.ErrCodeLessInstance})
				return
			}
			sessCtx.SetService(req.Service, srv)
			sess.SendMessage(&codec.HandshakeRes{Code: codec.ErrCodeSuccess})
			return
		}
	})
}
