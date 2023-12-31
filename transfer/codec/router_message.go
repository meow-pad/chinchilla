package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/meow-pad/chinchilla/transfer/common"
	"github.com/meow-pad/chinchilla/utils/codec"
	"io"
	"reflect"
)

type RpcRReq struct {
	SourceId string // 源请求服务id
	RPCId    uint32 // 源请求编号
	Payload  []byte
}

type RpcRRes struct {
	Code    uint16
	RPCId   uint32 // 源请求编号
	Payload []byte
}

func NewRpcRReq(SourceId string, rpcId uint32, payload []byte) *RpcRReq {
	return &RpcRReq{
		SourceId: SourceId,
		RPCId:    rpcId,
		Payload:  payload,
	}
}

func NewRpcRRes(rpcId uint32, payload []byte) *RpcRRes {
	return &RpcRRes{
		Code:    common.ErrCodeSuccess,
		RPCId:   rpcId,
		Payload: payload,
	}
}

func NewMessageRouter(byteOrder binary.ByteOrder,
	routerType int16, routerId string, routerMessage any) (*MessageRouter, error) {
	payload, err := encodeRouterMessage(byteOrder, routerMessage)
	if err != nil {
		return nil, err
	}
	return &MessageRouter{
		RouterType: routerType,
		RouterId:   routerId,
		Payload:    payload,
	}, nil
}

func encodeRouterMessage(byteOrder binary.ByteOrder, msg any) (buf []byte, err error) {
	switch req := msg.(type) {
	case *RpcRReq:
		buf = make([]byte, 1+2+len(req.SourceId)+4+len(req.Payload))
		buf[0] = TypeRPCRReq
		left := buf[1:]
		left, err = codec.WriteString(byteOrder, req.SourceId, left)
		if err != nil {
			return
		}
		left, err = codec.WriteUint32(byteOrder, req.RPCId, left)
		if err != nil {
			return
		}
		copy(left, req.Payload)
		return
	case *RpcRRes:
		buf = make([]byte, 1+2+4+len(req.Payload))
		buf[0] = TypeRPCRRes
		left := buf[1:]
		left, err = codec.WriteUint16(byteOrder, req.Code, left)
		left, err = codec.WriteUint32(byteOrder, req.RPCId, left)
		if err != nil {
			return
		}
		copy(left, req.Payload)
		return
	default:
		err = errors.New("unknown router message type:" + reflect.TypeOf(msg).String())
		return
	}
}

func decodeRouterMessage(byteOrder binary.ByteOrder, in []byte) (any, error) {
	if len(in) < 1 {
		return nil, io.ErrShortBuffer
	}
	msgType := in[0]
	switch msgType {
	case TypeRPCRReq:
		res := &RpcRReq{}
		left := in[1:]
		err := error(nil)
		if res.SourceId, left, err = codec.ReadString(byteOrder, left); err != nil {
			return nil, err
		}
		if res.RPCId, left, err = codec.ReadUint32(byteOrder, left); err != nil {
			return nil, err
		}
		res.Payload = bytes.Clone(left)
		return res, nil
	case TypeRPCRRes:
		res := &RpcRRes{}
		left := in[1:]
		err := error(nil)
		if res.Code, left, err = codec.ReadUint16(byteOrder, left); err != nil {
			return nil, err
		}
		if res.RPCId, left, err = codec.ReadUint32(byteOrder, left); err != nil {
			return nil, err
		}
		res.Payload = bytes.Clone(left)
		return res, nil
	default:
		return nil, fmt.Errorf("invalid message type:%d", msgType)
	}
}
