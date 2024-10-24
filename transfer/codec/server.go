package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/meow-pad/chinchilla/utils/codec"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"io"
	"reflect"
)

func NewServerCodec(order binary.ByteOrder) *ServerCodec {
	return &ServerCodec{byteOrder: order}
}

type ServerCodec struct {
	byteOrder binary.ByteOrder
}

func (sCodec *ServerCodec) Encode(msg any) ([]byte, error) {
	switch sMsg := msg.(type) {
	case *MessageSRes:
		buf := make([]byte, len(sMsg.Payload)+8+1)
		buf[0] = TypeMessageS
		sCodec.byteOrder.PutUint64(buf[1:], sMsg.ConnId)
		copy(buf[9:], sMsg.Payload)
		return buf, nil
	case *MessageRouter:
		buf := make([]byte, len(sMsg.Payload)+2+len(sMsg.RouterService)+2+len(sMsg.RouterId)+2+1)
		buf[0] = TypeMessageRouter
		left := buf[1:]
		err := error(nil)
		if left, err = codec.WriteString(sCodec.byteOrder, sMsg.RouterService, left); err != nil {
			return nil, err
		}
		if left, err = codec.WriteInt16(sCodec.byteOrder, sMsg.RouterType, left); err != nil {
			return nil, err
		}
		if left, err = codec.WriteString(sCodec.byteOrder, sMsg.RouterId, left); err != nil {
			return nil, err
		}
		copy(left, sMsg.Payload)
		return buf, nil
	case *BroadcastSRes:
		buf := make([]byte, codec.Uint64ArrayLen(sMsg.ConnIds)+len(sMsg.Payload)+1)
		buf[0] = TypeBroadcastS
		left := buf[1:]
		err := error(nil)
		if left, err = codec.WriteUint64Array(sCodec.byteOrder, sMsg.ConnIds, left); err != nil {
			return nil, err
		}
		copy(left, sMsg.Payload)
		return buf, nil
	case *RegisterSRes:
		buf := make([]byte, len(sMsg.Payload)+8+2+2+len(sMsg.RouterId)+1)
		buf[0] = TypeRegisterS
		left := buf[1:]
		err := error(nil)
		if left, err = codec.WriteUint64(sCodec.byteOrder, sMsg.ConnId, left); err != nil {
			return nil, err
		}
		if left, err = codec.WriteUint16(sCodec.byteOrder, sMsg.Code, left); err != nil {
			return nil, err
		}
		if left, err = codec.WriteString(sCodec.byteOrder, sMsg.RouterId, left); err != nil {
			return nil, err
		}
		copy(left, sMsg.Payload)
		return buf, nil
	case *UnregisterSRes:
		plog.Debug("encode UnregisterSRes", pfield.Uint64("connId", sMsg.ConnId), pfield.Stack("stack"))
		buf := make([]byte, 8+1)
		buf[0] = TypeUnregisterS
		left := buf[1:]
		err := error(nil)
		if left, err = codec.WriteUint64(sCodec.byteOrder, sMsg.ConnId, left); err != nil {
			return nil, err
		}
		return buf, nil
	case *HeartbeatSRes:
		//plog.Debug("encode HeartbeatSRes", pfield.Uint64("connId", sMsg.ConnId), pfield.Stack("stack"))
		buf := make([]byte, len(sMsg.Payload)+8+1)
		buf[0] = TypeHeartbeatS
		sCodec.byteOrder.PutUint64(buf[1:], sMsg.ConnId)
		copy(buf[9:], sMsg.Payload)
		return buf, nil
	case *HandshakeRes:
		buf := make([]byte, 2+1)
		buf[0] = TypeHandshake
		sCodec.byteOrder.PutUint16(buf[1:], sMsg.Code)
		return buf, nil
	case *ServiceInstIReq:
		buf := make([]byte, 2+len(sMsg.ServiceName)+1)
		buf[0] = TypeServiceInstIReq
		left := buf[1:]
		err := error(nil)
		if left, err = codec.WriteString(sCodec.byteOrder, sMsg.ServiceName, left); err != nil {
			return nil, err
		}
		return buf, nil
	case *SegmentMsg:
		return encodeSegmentMsg(sCodec.byteOrder, sMsg)
	case *RpcRReq, *RpcRRes:
		// 这些消息不可能在server端编码
		return nil, errors.New("unsupported message in server encoder:" + reflect.TypeOf(msg).String())
	default:
		return nil, errors.New("(transfer server) encode invalid message type:" + reflect.TypeOf(msg).String())
	}
}

func (sCodec *ServerCodec) Decode(in []byte) (any, error) {
	inLen := len(in)
	if inLen < 1 {
		return nil, io.ErrShortBuffer
	}
	msgType := in[0]
	switch msgType {
	case TypeMessageS:
		req := &MessageSReq{}
		left := in[1:]
		err := error(nil)
		if req.ConnId, left, err = codec.ReadUint64(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		req.Payload = bytes.Clone(left)
		return req, nil
	case TypeRPCRReq, TypeRPCRRes:
		return decodeRouterMessage(sCodec.byteOrder, in)
	case TypeRegisterS:
		req := &RegisterSReq{}
		left := in[1:]
		err := error(nil)
		if req.ConnId, left, err = codec.ReadUint64(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		req.Payload = bytes.Clone(left)
		return req, nil
	case TypeUnregisterS:
		req := &UnregisterSReq{}
		left := in[1:]
		err := error(nil)
		if req.ConnId, left, err = codec.ReadUint64(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		return req, nil
	case TypeHeartbeatS:
		req := &HeartbeatSReq{}
		left := in[1:]
		err := error(nil)
		if req.ConnId, left, err = codec.ReadUint64(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		req.Payload = bytes.Clone(left)
		return req, nil
	case TypeHandshake:
		req := &HandshakeReq{}
		left := in[1:]
		err := error(nil)
		if req.Id, left, err = codec.ReadString(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if req.AuthKey, left, err = codec.ReadString(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if req.Service, left, err = codec.ReadString(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if req.ServiceId, left, err = codec.ReadString(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if req.ConnIds, left, err = codec.ReadUint64Array(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if req.RouterIds, left, err = codec.ReadStringArray(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		return req, nil
	case TypeServiceInstIReS:
		res := &ServiceInstIRes{}
		left := in[1:]
		err := error(nil)
		if res.Code, left, err = codec.ReadUint16(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if res.ServiceName, left, err = codec.ReadString(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if res.ServiceInstArr, left, err = codec.ReadStringArray(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		return res, nil
	case TypeSegment:
		return decodeSegmentMsg(sCodec.byteOrder, in[1:])
	case TypeMessageRouter:
		// 这些消息不可能在server端解码
		return nil, fmt.Errorf("unsupported message in server decoder:%d", msgType)
	default:
		return nil, fmt.Errorf("(transfer server) decode invalid message type:%d", msgType)
	}
}
