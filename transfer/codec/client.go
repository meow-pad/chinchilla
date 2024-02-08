package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/meow-pad/chinchilla/utils/codec"
	"io"
	"reflect"
)

func NewClientCodec(order binary.ByteOrder) *ClientCodec {
	return &ClientCodec{byteOrder: order}
}

type ClientCodec struct {
	byteOrder binary.ByteOrder
}

func (cCodec *ClientCodec) Encode(msg any) ([]byte, error) {
	switch cMsg := msg.(type) {
	case *MessageSReq:
		buf := make([]byte, len(cMsg.Payload)+8+1)
		buf[0] = TypeMessageS
		cCodec.byteOrder.PutUint64(buf[1:], cMsg.ConnId)
		copy(buf[9:], cMsg.Payload)
		return buf, nil
	case []byte: // 直接转发的消息数据
		return cMsg, nil
	case *RegisterSReq:
		buf := make([]byte, len(cMsg.Payload)+8+1)
		buf[0] = TypeRegisterS
		cCodec.byteOrder.PutUint64(buf[1:], cMsg.ConnId)
		copy(buf[9:], cMsg.Payload)
		return buf, nil
	case *UnregisterSReq:
		buf := make([]byte, 8+1)
		buf[0] = TypeUnregisterS
		cCodec.byteOrder.PutUint64(buf[1:], cMsg.ConnId)
		return buf, nil
	case *HeartbeatSReq:
		buf := make([]byte, len(cMsg.Payload)+8+1)
		buf[0] = TypeHeartbeatS
		cCodec.byteOrder.PutUint64(buf[1:], cMsg.ConnId)
		copy(buf[9:], cMsg.Payload)
		return buf, nil
	case *HandshakeReq:
		buf := make([]byte, 1+8+len(cMsg.Id)+len(cMsg.AuthKey)+len(cMsg.Service)+len(cMsg.ServiceId)+
			codec.Uint64ArrayLen(cMsg.ConnIds)+codec.StringArrayLen(cMsg.RouterIds))
		buf[0] = TypeHandshake
		left := buf[1:]
		err := error(nil)
		left, err = codec.WriteString(cCodec.byteOrder, cMsg.Id, left)
		if err != nil {
			return nil, err
		}
		left, err = codec.WriteString(cCodec.byteOrder, cMsg.AuthKey, left)
		if err != nil {
			return nil, err
		}
		left, err = codec.WriteString(cCodec.byteOrder, cMsg.Service, left)
		if err != nil {
			return nil, err
		}
		left, err = codec.WriteString(cCodec.byteOrder, cMsg.ServiceId, left)
		if err != nil {
			return nil, err
		}
		left, err = codec.WriteUint64Array(cCodec.byteOrder, cMsg.ConnIds, left)
		if err != nil {
			return nil, err
		}
		left, err = codec.WriteStringArray(cCodec.byteOrder, cMsg.RouterIds, left)
		if err != nil {
			return nil, err
		}
		return buf, nil
	case *ServiceInstIRes:
		buf := make([]byte, 2+2+len(cMsg.ServiceName)+codec.StringArrayLen(cMsg.ServiceInstArr)+1)
		buf[0] = TypeServiceInstIReS
		left := buf[1:]
		err := error(nil)
		left, err = codec.WriteUint16(cCodec.byteOrder, cMsg.Code, left)
		if err != nil {
			return nil, err
		}
		left, err = codec.WriteString(cCodec.byteOrder, cMsg.ServiceName, left)
		if err != nil {
			return nil, err
		}
		left, err = codec.WriteStringArray(cCodec.byteOrder, cMsg.ServiceInstArr, left)
		if err != nil {
			return nil, err
		}
		return buf, nil
	case *SegmentMsg:
		return encodeSegmentMsg(cCodec.byteOrder, cMsg)
	case *MessageRouter, *RpcRReq, *RpcRRes:
		// 这些消息不可能在client端编码
		return nil, errors.New("unsupported message in client encoder:" + reflect.TypeOf(msg).String())
	default:
		return nil, errors.New("invalid message type:" + reflect.TypeOf(msg).String())
	}
}

func (cCodec *ClientCodec) Decode(in []byte) (any, error) {
	inLen := len(in)
	if inLen < 1 {
		return nil, io.ErrShortBuffer
	}
	msgType := in[0]
	switch msgType {
	case TypeMessageS:
		res := &MessageSRes{}
		left := in[1:]
		err := error(nil)
		if res.ConnId, left, err = codec.ReadUint64(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		res.Payload = bytes.Clone(left)
		return res, nil
	case TypeMessageRouter:
		res := &MessageRouter{}
		left := in[1:]
		err := error(nil)
		if res.RouterService, left, err = codec.ReadString(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if res.RouterType, left, err = codec.ReadInt16(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if res.RouterId, left, err = codec.ReadString(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		res.Payload = bytes.Clone(left)
		return res, nil
	case TypeBroadcastS:
		res := &BroadcastSRes{}
		left := in[1:]
		err := error(nil)
		if res.ConnIds, left, err = codec.ReadUint64Array(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		res.Payload = bytes.Clone(left)
		return res, nil
	case TypeRegisterS:
		res := &RegisterSRes{}
		left := in[1:]
		err := error(nil)
		if res.ConnId, left, err = codec.ReadUint64(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if res.Code, left, err = codec.ReadUint16(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if res.RouterId, left, err = codec.ReadString(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		res.Payload = bytes.Clone(left)
		return res, nil
	case TypeUnregisterS:
		res := &UnregisterSRes{}
		left := in[1:]
		err := error(nil)
		if res.ConnId, left, err = codec.ReadUint64(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		return res, nil
	case TypeHeartbeatS:
		res := &HeartbeatSRes{}
		left := in[1:]
		err := error(nil)
		if res.ConnId, left, err = codec.ReadUint64(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		res.Payload = bytes.Clone(left)
		return res, nil
	case TypeHandshake:
		res := &HandshakeRes{}
		left := in[1:]
		err := error(nil)
		if res.Code, left, err = codec.ReadUint16(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		return res, nil
	case TypeServiceInstIReq:
		req := &ServiceInstIReq{}
		left := in[1:]
		err := error(nil)
		if req.ServiceName, left, err = codec.ReadString(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		return req, nil
	case TypeSegment:
		return decodeSegmentMsg(cCodec.byteOrder, in[1:])
	case TypeRPCRReq, TypeRPCRRes:
		// 这些消息不可能在client端解码
		return nil, fmt.Errorf("unsupported message in client decoder:%d", msgType)
	default:
		return nil, errors.New("invalid message type")
	}
}
