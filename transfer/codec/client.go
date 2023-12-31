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
	switch req := msg.(type) {
	case *MessageSReq:
		buf := make([]byte, len(req.Payload)+8+1)
		buf[0] = TypeMessageS
		cCodec.byteOrder.PutUint64(buf[1:], req.ConnId)
		copy(buf[9:], req.Payload)
		return buf, nil
	case []byte: // 直接转发的消息数据
		return req, nil
	case *RegisterSReq:
		buf := make([]byte, len(req.Payload)+8+1)
		buf[0] = TypeRegisterS
		cCodec.byteOrder.PutUint64(buf[1:], req.ConnId)
		copy(buf[9:], req.Payload)
		return buf, nil
	case *UnregisterSReq:
		buf := make([]byte, 8+1)
		buf[0] = TypeUnregisterS
		cCodec.byteOrder.PutUint64(buf[1:], req.ConnId)
		return buf, nil
	case *HeartbeatSReq:
		buf := make([]byte, len(req.Payload)+8+1)
		buf[0] = TypeHeartbeatS
		cCodec.byteOrder.PutUint64(buf[1:], req.ConnId)
		copy(buf[9:], req.Payload)
		return buf, nil
	case *HandshakeReq:
		buf := make([]byte, 1+8+len(req.Id)+len(req.AuthKey)+len(req.Service)+len(req.ServiceId)+
			codec.Uint64ArrayLen(req.ConnIds)+codec.Uint64ArrayLen(req.RouterIds))
		buf[0] = TypeHandshake
		left := buf[1:]
		err := error(nil)
		left, err = codec.WriteString(cCodec.byteOrder, req.Id, left)
		if err != nil {
			return nil, err
		}
		left, err = codec.WriteString(cCodec.byteOrder, req.AuthKey, left)
		if err != nil {
			return nil, err
		}
		left, err = codec.WriteString(cCodec.byteOrder, req.Service, left)
		if err != nil {
			return nil, err
		}
		left, err = codec.WriteString(cCodec.byteOrder, req.ServiceId, left)
		if err != nil {
			return nil, err
		}
		left, err = codec.WriteUint64Array(cCodec.byteOrder, req.ConnIds, left)
		if err != nil {
			return nil, err
		}
		left, err = codec.WriteUint64Array(cCodec.byteOrder, req.RouterIds, left)
		if err != nil {
			return nil, err
		}
		return buf, nil
	case *SegmentMsg:
		return encodeSegmentMsg(cCodec.byteOrder, req)
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
		if res.RouterType, left, err = codec.ReadInt16(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if res.RouterId, left, err = codec.ReadString(cCodec.byteOrder, left); err != nil {
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
		if res.RouterId, left, err = codec.ReadUint64(cCodec.byteOrder, left); err != nil {
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
	case TypeSegment:
		return decodeSegmentMsg(cCodec.byteOrder, in[1:])
	case TypeRPCRReq, TypeRPCRRes:
		// 这些消息不可能在client端解码
		return nil, fmt.Errorf("unsupported message in client decoder:%d", msgType)
	default:
		return nil, errors.New("invalid message type")
	}
}
