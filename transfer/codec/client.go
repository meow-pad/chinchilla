package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
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
		buf := make([]byte, 1+8+len(req.Id)+len(req.AuthKey)+len(req.Service)+len(req.ServiceId))
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
		return buf, nil
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
	default:
		return nil, errors.New("invalid message type")
	}
}
