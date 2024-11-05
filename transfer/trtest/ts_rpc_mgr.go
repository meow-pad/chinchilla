package trtest

import (
	"fmt"
	"github.com/meow-pad/chinchilla/transfer/codec"
	"github.com/meow-pad/chinchilla/transfer/router"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"github.com/meow-pad/persian/utils/coding"
	"github.com/meow-pad/persian/utils/collections"
	"reflect"
	"sync/atomic"
)

func newTSRPCManager(runtime *TransferRuntime, server *TransferServer) *TSRPCManager {
	return &TSRPCManager{
		Runtime: runtime,
		Server:  server,
	}
}

type TSRPCManager struct {
	Runtime *TransferRuntime
	Server  *TransferServer

	rpcHandler     TsRPCHandler
	rpcIdGenerator atomic.Uint32
	rpcHandlers    collections.SyncMap[uint32, func(msg Message)]
}

func (manager *TSRPCManager) SendRPCRequest(sess session.Session, targetService, targetServiceId string, msg Message, respHandler func(msg Message)) {
	msgBytes, err := rpcMsgCodec.Encode(msg)
	if err != nil {
		respHandler(&RPCErrResp{
			Err: fmt.Sprintf("encode rpc msg error:%s", err),
		})
		return
	}
	rpcId := manager.rpcIdGenerator.Add(1)

	// 构造路由消息
	sourceSrv := manager.Server.Options.ServiceName
	sourceSrvId := manager.Server.Options.ServiceId
	routerMsg := codec.NewRpcRReq(sourceSrv, sourceSrvId, rpcId, msgBytes)
	routerSrv := targetService
	msgRouter, err := codec.NewMessageRouter(codec.MessageCodecByteOrder, routerSrv,
		router.RouteTypeService, targetServiceId, manager.Server.MsgCoder(), routerMsg)
	if err != nil {
		respHandler(&RPCErrResp{
			Err: fmt.Sprintf("construct rpc router msg error:%s", err),
		})
		return
	}
	manager.rpcHandlers.Store(rpcId, respHandler)
	manager.Runtime.Timer.AfterFunc(RPCTimeout, func() {
		dRespHandler, _ := manager.rpcHandlers.Delete(rpcId)
		if dRespHandler != nil {
			dRespHandler(&RPCErrResp{
				Err: "rpc timeout",
			})
		}
	})
	sess.SendMessage(msgRouter)
}

func (manager *TSRPCManager) HandleRPCRequest(sess session.Session, req *codec.RpcRReq) (noReturn error) {
	tCtx := coding.Cast[*RemoteContext](sess.Context())
	if tCtx == nil {
		manager.sendRouterMessage(sess, req.SourceSrv, req.SourceId, &codec.RpcRRes{
			Code: ServerCodeSessionLessContext,
		})
		plog.Error("invalid transfer sess context:",
			pfield.String("context type", reflect.TypeOf(sess.Context()).String()))
		return nil
	}
	if !tCtx.IsHandShook() {
		plog.Error("handshake first")
		manager.sendRouterMessage(sess, req.SourceSrv, req.SourceId, &codec.RpcRRes{
			Code: ServerCodeSessionHandshakeFirst,
		})
		return nil
	}
	var respMsg Message
	reqMsg, err := rpcMsgCodec.Decode(req.Payload)
	if err != nil {
		respMsg = &RPCErrResp{
			Err: fmt.Sprintf("decode rpc msg error:%s", err),
		}
	} else {
		respMsg, err = manager.rpcHandler.handleRequest(reqMsg)
		if err != nil {
			respMsg = &RPCErrResp{
				Err: fmt.Sprintf("handle rpc msg error:%s", err),
			}
		} else {
			if respMsg == nil {
				respMsg = &RPCErrResp{
					Err: "",
				}
			}
		} // end of else
	}

	if respBytes, eErr := rpcMsgCodec.Encode(respMsg); eErr != nil {
		plog.Error("encode rpc response error:", pfield.Error(eErr))
	} else {
		// 构造路由消息
		routerMsg := codec.NewRpcRRes(req.RPCId, respBytes)
		manager.sendRouterMessage(sess, req.SourceSrv, req.SourceId, routerMsg)
	}
	return
}

func (manager *TSRPCManager) sendRouterMessage(sess session.Session, sourceSrv, sourceServiceId string, routerMsg any) {
	msgRouter, rErr := codec.NewMessageRouter(codec.MessageCodecByteOrder, sourceSrv,
		router.RouteTypeService, sourceServiceId, manager.Server.MsgCoder(), routerMsg)
	if rErr != nil {
		plog.Error("construct rpc router msg error:", pfield.Error(rErr))
	}
	sess.SendMessage(msgRouter)
}

func (manager *TSRPCManager) HandleRPCResponse(sess session.Session, msg *codec.RpcRRes) (noReturn error) {
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
	rpcId := msg.RPCId
	respHandler, _ := manager.rpcHandlers.Delete(rpcId)
	if respHandler == nil {
		plog.Error("rpc response handler not found",
			pfield.Uint32("rpcId", rpcId),
			pfield.ByteString("payload", msg.Payload),
		)
		return
	}
	if msg.Code != RpcCodeSuccess {
		respHandler(&RPCErrResp{
			Err: fmt.Sprintf("rpc error code:%d", msg.Code),
		})
	} else {
		respMsg, err := rpcMsgCodec.Decode(msg.Payload)
		if err != nil {
			respHandler(&RPCErrResp{
				Err: fmt.Sprintf("decode rpc response error:%s", err)})
		} else {
			plog.Info("handle rpc response",
				pfield.Uint32("rpcId", rpcId),
				pfield.ByteString("payload", msg.Payload),
			)
			respHandler(respMsg)
		}
	}
	return
}
