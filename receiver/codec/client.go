package codec

import (
	"errors"
	"io"
	"reflect"
)

type ClientCodec struct {
}

func (cCodec *ClientCodec) Encode(msg any) ([]byte, error) {
	switch req := msg.(type) {
	case Message:
		size := req.XXX_Size()
		buf := make([]byte, size+1)
		buf[0] = req.Type()
		err := error(nil)
		_, err = req.XXX_Marshal(buf[1:1], false)
		if err != nil {
			return nil, err
		}
		return buf, nil
	default:
		return nil, errors.New("(receiver client) invalid message type:" + reflect.TypeOf(msg).String())
	}
}

func (cCodec *ClientCodec) Decode(in []byte) (any, error) {
	inLen := len(in)
	if inLen < 1 {
		return nil, io.ErrShortBuffer
	}
	msgType := in[0]
	msg, err := newResMessage(msgType)
	if err != nil {
		return nil, err
	}
	err = msg.XXX_Unmarshal(in[1:])
	if err != nil {
		return nil, err
	}
	return msg, nil
}
