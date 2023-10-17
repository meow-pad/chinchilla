package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/meow-pad/chinchilla/utils/codec"
	"github.com/meow-pad/persian/frame/pnet"
	"io"
	"reflect"
)

type ClientCodec struct {
	byteOrder binary.ByteOrder
}

func (cCodec *ClientCodec) Encode(msg any) ([]byte, error) {
	switch req := msg.(type) {
	case *MessageReq:
		lenSrv := len(req.Service)
		if lenSrv > maxServiceLen {
			return nil, pnet.ErrMessageTooLarge
		}
		buf := make([]byte, len(req.Payload)+(lenSrv+2)+1)
		buf[0] = TypeMessage
		left := buf[1:]
		err := error(nil)
		if left, err = codec.WriteString(cCodec.byteOrder, req.Service, left); err != nil {
			return nil, err
		}
		copy(left, req.Payload)
		return buf, nil
	case *HeartbeatReq:
		buf := make([]byte, len(req.Payload)+1)
		buf[0] = TypeHeartbeat
		left := buf[1:]
		copy(left, req.Payload)
		return buf, nil
	case *HandshakeReq:
		buf := make([]byte, len(req.RouterId)+len(req.AuthKey)+len(req.Service)+6+1)
		buf[0] = TypeHandshake
		left := buf[1:]
		err := error(nil)
		if left, err = codec.WriteString(cCodec.byteOrder, req.RouterId, left); err != nil {
			return nil, err
		}
		if left, err = codec.WriteString(cCodec.byteOrder, req.AuthKey, left); err != nil {
			return nil, err
		}
		if left, err = codec.WriteString(cCodec.byteOrder, req.Service, left); err != nil {
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
	case TypeMessage:
		res := &MessageRes{}
		left := in[1:]
		err := error(nil)
		if res.Code, left, err = codec.ReadUint16(cCodec.byteOrder, left); err != nil {
			return nil, err
		}
		res.Payload = bytes.Clone(left)
		return res, nil
	case TypeHeartbeat:
		res := &HeartbeatRes{}
		left := in[1:]
		err := error(nil)
		if res.Code, left, err = codec.ReadUint16(cCodec.byteOrder, left); err != nil {
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
