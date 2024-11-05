package trtest

import (
	"fmt"
	"github.com/meow-pad/chinchilla/transfer/codec"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"github.com/meow-pad/persian/utils/coding"
	"reflect"
	"time"
)

func newTSHandler(server *TransferServer, rpcMgr *TSRPCManager) *TSHandler {
	return &TSHandler{
		Server: server,
		RPCMgr: rpcMgr,
	}
}

type TSHandler struct {
	Server *TransferServer
	RPCMgr *TSRPCManager

	msgHandler TsMsgHandler
}

func (handler *TSHandler) HandleMessage(sess session.Session, msg any) error {
	switch req := msg.(type) {
	case *codec.MessageSReq:
		return handler.handleMessagesReq(sess, req)
	case *codec.RpcRReq:
		return handler.RPCMgr.HandleRPCRequest(sess, req)
	case *codec.RpcRRes:
		return handler.RPCMgr.HandleRPCResponse(sess, req)
	case *codec.RegisterSReq:
		return handler.handleRegistersReq(sess, req)
	case *codec.UnregisterSReq:
		return handler.handleUnregistersReq(sess, req)
	case *codec.HeartbeatSReq:
		return handler.handleHeartbeatReq(sess, req)
	case *codec.HandshakeReq:
		return handler.handleHandshakeReq(sess, req)
	case *codec.ServiceInstIRes:
		return handler.HandleServiceInstRes(sess, req)
	case *codec.SegmentMsg:
		return handler.handleSegmentMsg(sess, req)
	default:
		plog.Error("unknown msg type", pfield.String("type", reflect.TypeOf(msg).String()))
	}
	return nil
}

func (handler *TSHandler) handleRegistersReq(sess session.Session, req *codec.RegisterSReq) error {
	tCtx := coding.Cast[*RemoteContext](sess.Context())
	if tCtx == nil {
		sess.SendMessage(&codec.RegisterSRes{
			Code: ServerCodeSessionLessContext,
		})
		plog.Error("invalid transfer sess context:",
			pfield.String("context type", reflect.TypeOf(sess.Context()).String()))
		return nil
	}
	if !tCtx.IsHandShook() {
		plog.Error("handshake first")
		sess.SendMessage(&codec.RegisterSRes{
			Code: ServerCodeSessionHandshakeFirst,
		})
		return nil
	}
	uSess := handler.Server.userMgr.GetUserSession(req.ConnId)
	if uSess == nil {
		// 构建一个包含底层会话的空包装器
		uSess = &UserSession{
			connId: req.ConnId,
		}
		handler.Server.userMgr.SetUserSession(req.ConnId, uSess)
	}
	if len(req.Payload) <= 0 {
		// 非法的请求
		return nil
	}
	msg, err := msgCodec.Decode(req.Payload)
	if err != nil {
		return err
	}
	switch tMsg := msg.(type) {
	case *LoginReq:
		if uSess.uid.Load() != UIDInvalid {
			if uSess.uid.Load() != tMsg.Uid {
				sess.SendMessage(&codec.RegisterSRes{
					ConnId: req.ConnId,
					Code:   UserCodeRepeatLogin,
				})
				return nil
			}
		}
		plog.Info("ts handle RegisterSReq(LoginReq):",
			pfield.Uint64("connId", req.ConnId),
			pfield.ByteString("msg", req.Payload),
		)
		handler.Server.userMgr.AddUser(tMsg.Uid, uSess)
	default:
		// 返回错误码，让用户重连
		plog.Error("unsupported message type in RegistersReq,login first",
			pfield.Uint64("connId", req.ConnId),
			pfield.ByteString("msg", req.Payload),
		)
		sess.SendMessage(&codec.RegisterSRes{
			ConnId: req.ConnId,
			Code:   UserCodeAuthFailed,
		})
	}
	return nil
}

func (handler *TSHandler) handleUnregistersReq(sess session.Session, req *codec.UnregisterSReq) error {
	tCtx := coding.Cast[*RemoteContext](sess.Context())
	if tCtx == nil {
		plog.Error("invalid transfer sess context:",
			pfield.String("context type", reflect.TypeOf(sess.Context()).String()))
		return nil
	}
	if !tCtx.IsHandShook() {
		plog.Error("handshake first")
		return nil
	}
	if req.ConnId == 0 {
		return nil
	}
	// 用户下线
	uSess := handler.Server.userMgr.GetUserSession(req.ConnId)
	if uSess == nil {
		return nil
	}
	handler.Server.userMgr.RemoveUserSession(uSess.connId)
	sess.SendMessage(&codec.UnregisterSRes{
		ConnId: req.ConnId,
	})
	return nil
}

func (handler *TSHandler) handleHeartbeatReq(sess session.Session, req *codec.HeartbeatSReq) error {
	tCtx := coding.Cast[*RemoteContext](sess.Context())
	if tCtx == nil {
		plog.Error("invalid transfer sess context:",
			pfield.String("context type", reflect.TypeOf(sess.Context()).String()))
		return nil
	}
	if !tCtx.IsHandShook() {
		plog.Error("handshake first")
		return nil
	}
	uSess := handler.Server.userMgr.GetUserSession(req.ConnId)
	if uSess == nil {
		plog.Debug("close less user session",
			pfield.Uint64("connId", req.ConnId),
			pfield.String("svrInstId", handler.Server.ServiceId()),
			pfield.Stack("stack"))
		return nil
	}
	msg, err := msgCodec.Decode(req.Payload)
	if err != nil {
		return err
	}
	msgReq := msg.(*HeartbeatReq)
	if msgReq == nil {
		plog.Warn("invalid msg in heartbeat request", pfield.Uint16("msgType", msg.Type()))
		return nil
	}
	msgResp := &HeartbeatResp{
		ReqId: msgReq.ReqId,
		CTime: msgReq.CTime,
		STime: time.Now().UnixMilli(),
	}
	payload, eErr := msgCodec.Encode(msgResp)
	if eErr != nil {
		plog.Error("encode heartbeat response error:", pfield.Error(eErr))
		return eErr
	}
	sess.SendMessage(&codec.HeartbeatSRes{
		ConnId:  req.ConnId,
		Payload: payload,
	})
	return nil
}

func (handler *TSHandler) handleMessagesReq(sess session.Session, req *codec.MessageSReq) error {
	tCtx := coding.Cast[*RemoteContext](sess.Context())
	if tCtx == nil {
		plog.Error("invalid transfer sess context:",
			pfield.String("context type", reflect.TypeOf(sess.Context()).String()))
		return nil
	}
	if !tCtx.IsHandShook() {
		plog.Error("handshake first")
		return nil
	}
	uSess := handler.Server.userMgr.GetUserSession(req.ConnId)
	if uSess == nil {
		plog.Warn("close less user session",
			pfield.Uint64("connId", req.ConnId),
			pfield.Stack("stack"))
		// 网关认为已注册，但实际不存在的用户，通知断开连接
		sess.SendMessage(&codec.UnregisterSRes{
			ConnId: req.ConnId,
		})
		return nil
	}
	if uSess.uid.Load() == UIDInvalid {
		plog.Error("login first:",
			pfield.Uint64("connId", req.ConnId),
		)
		return nil
	}
	msg, err := msgCodec.Decode(req.Payload)
	if err != nil {
		return err
	}
	err = handler.msgHandler.handle(uSess, msg)
	if err != nil {
		plog.Error("handle user msg error:", pfield.Error(err))
	} else {
		uSess.Access()
	}
	return nil
}

func (handler *TSHandler) handleHandshakeReq(sess session.Session, req *codec.HandshakeReq) error {
	// 校验
	if len(req.Id) <= 0 {
		sess.SendMessage(&codec.HandshakeRes{
			Code: ServerCodeInvalidTransferId,
		})
		return nil
	}
	if req.AuthKey != ServerHandshakeAuth {
		sess.SendMessage(&codec.HandshakeRes{
			Code: ServerCodeInvalidAuth,
		})
		return nil
	}
	if req.Service != handler.Server.ServiceName() {
		sess.SendMessage(&codec.HandshakeRes{
			Code: ServerCodeInvalidService,
		})
		return nil
	}
	if req.ServiceId != handler.Server.ServiceId() {
		sess.SendMessage(&codec.HandshakeRes{
			Code: ServerCodeInvalidServiceId,
		})
		return nil
	}
	// 完成握手
	tCtx := coding.Cast[*RemoteContext](sess.Context())
	if tCtx == nil {
		sess.SendMessage(&codec.HandshakeRes{
			Code: ServerCodeSessionLessContext,
		})
		plog.Error("invalid transfer sess context:",
			pfield.String("context type", reflect.TypeOf(sess.Context()).String()))
		return nil
	}
	tCtx.ShakeHand(req.Id)
	sess.SendMessage(&codec.HandshakeRes{})
	handler.Server.sessMgr.AddSess(req.Id, sess)
	plog.Debug("=(transfer server handler) handshake with client",
		pfield.String("serviceId", handler.Server.ServiceId()),
		pfield.String("client", sess.Connection().RemoteAddr().String()),
		pfield.String("remoteServiceId", req.Id),
	)
	return nil
}

func (handler *TSHandler) handleSegmentMsg(session session.Session, msg *codec.SegmentMsg) error {
	return fmt.Errorf("unsupported SegmentMsg")
}

func (handler *TSHandler) HandleServiceInstRes(sess session.Session, msg *codec.ServiceInstIRes) error {
	return fmt.Errorf("unsupported ServiceInstIRes")
}
