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
	case *MessageRes:
		buf := make([]byte, len(res.Payload)+2+1)
		buf[0] = TypeMessage
		left := buf[1:]
		err := error(nil)
		left, err = codec.WriteUint16(sCodec.byteOrder, res.Code, left)
		if err != nil {
			return nil, err
		}
		copy(left, res.Payload)
		return buf, nil
	case *HeartbeatRes:
		buf := make([]byte, len(res.Payload)+2+1)
		buf[0] = TypeHeartbeat
		left := buf[1:]
		err := error(nil)
		left, err = codec.WriteUint16(sCodec.byteOrder, res.Code, left)
		if err != nil {
			return nil, err
		}
		copy(left, res.Payload)
		return buf, nil
	case *HandshakeRes:
		buf := make([]byte, 3)
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
	case TypeMessage:
		req := &MessageReq{}
		left := in[1:]
		err := error(nil)
		if req.Service, left, err = codec.ReadString(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		req.Payload = bytes.Clone(left)
		return req, nil
	case TypeHeartbeat:
		req := &HeartbeatReq{}
		left := in[1:]
		req.Payload = bytes.Clone(left)
		return req, nil
	case TypeHandshake:
		req := &HandshakeReq{}
		left := in[1:]
		err := error(nil)
		if req.RouterId, left, err = codec.ReadString(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if req.AuthKey, left, err = codec.ReadString(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		if req.Service, left, err = codec.ReadString(sCodec.byteOrder, left); err != nil {
			return nil, err
		}
		return req, nil
	default:
		return nil, errors.New("invalid message type")
	}
}
