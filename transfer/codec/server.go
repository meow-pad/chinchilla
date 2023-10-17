package codec

import (
	"bytes"
	"chinchilla/utils/codec"
	"encoding/binary"
	"errors"
	"io"
	"reflect"
)

type ServerCodec struct {
	byteOrder binary.ByteOrder
}

func (sCodec *ServerCodec) Encode(msg any) ([]byte, error) {
	switch res := msg.(type) {
	case *MessageSRes:
		buf := make([]byte, len(res.Payload)+8+1)
		buf[0] = TypeMessageS
		sCodec.byteOrder.PutUint64(buf[1:], res.ConnId)
		copy(buf[9:], res.Payload)
		return buf, nil
	case *RegisterSRes:
		buf := make([]byte, len(res.Payload)+18+1)
		buf[0] = TypeRegisterS
		left := buf[1:]
		err := error(nil)
		if left, err = codec.WriteUint64(sCodec.byteOrder, res.ConnId, left); err != nil {
			return nil, err
		}
		if left, err = codec.WriteUint16(sCodec.byteOrder, res.Code, left); err != nil {
			return nil, err
		}
		if left, err = codec.WriteUint64(sCodec.byteOrder, res.RouterId, left); err != nil {
			return nil, err
		}
		copy(left, res.Payload)
		return buf, nil
	case *UnregisterSRes:
		buf := make([]byte, 8+1)
		buf[0] = TypeUnregisterS
		left := buf[1:]
		err := error(nil)
		if left, err = codec.WriteUint64(sCodec.byteOrder, res.ConnId, left); err != nil {
			return nil, err
		}
		return buf, nil
	case *HeartbeatSRes:
		buf := make([]byte, len(res.Payload)+8+1)
		buf[0] = TypeHeartbeatS
		sCodec.byteOrder.PutUint64(buf[1:], res.ConnId)
		copy(buf[9:], res.Payload)
		return buf, nil
	case *HandshakeRes:
		buf := make([]byte, 2+1)
		buf[0] = TypeHandshake
		sCodec.byteOrder.PutUint16(buf[1:], res.Code)
		return buf, nil
	default:
		return nil, errors.New("invalid message type:" + reflect.TypeOf(msg).String())
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
		return req, nil
	default:
		return nil, errors.New("invalid message type")
	}
}
