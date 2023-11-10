package codec

import (
	"encoding/binary"
	"github.com/stretchr/testify/require"
	"math/rand"
	"reflect"
	"testing"
)

func _getObjectValue(ptr any) any {
	ptrValue := reflect.ValueOf(ptr)
	return ptrValue.Elem().Interface()
}

func TestCodec_Req(t *testing.T) {
	should := require.New(t)
	handshakeReq := &HandshakeReq{
		RouterId: 1234,
		AuthKey:  "1234567",
		Service:  "test",
	}
	heartbeatReq := &HeartbeatReq{
		Payload: []byte{1, 2, 3, 4, 5},
	}
	messageReq := &MessageReq{
		Service: "test001",
		Payload: []byte{1, 2, 3, 4, 5},
	}
	messages := []any{handshakeReq, heartbeatReq, messageReq}
	cCodec := ClientCodec{byteOrder: binary.LittleEndian}
	sCodec := ServerCodec{byteOrder: binary.LittleEndian}
	for _, msg := range messages {
		data, err := cCodec.Encode(msg)
		should.Nil(err)
		dMsg, err := sCodec.Decode(data)
		should.Nil(err)
		// 将原字节乱序
		rand.Shuffle(len(data), func(i, j int) {
			data[i], data[j] = data[j], data[i]
		})
		should.Equal(_getObjectValue(msg), _getObjectValue(dMsg))
	}
}

func TestCodec_Res(t *testing.T) {
	should := require.New(t)
	handshakeRes := &HandshakeRes{
		Code: 123,
	}
	heartbeatRes := &HeartbeatRes{
		Code:    123,
		Payload: []byte{1, 2, 3, 4, 5},
	}
	messageRes := &MessageRes{
		Code:    123,
		Payload: []byte{1, 2, 3, 4, 5},
	}
	messages := []any{handshakeRes, heartbeatRes, messageRes}
	cCodec := ClientCodec{byteOrder: binary.LittleEndian}
	sCodec := ServerCodec{byteOrder: binary.LittleEndian}
	for _, msg := range messages {
		data, err := sCodec.Encode(msg)
		should.Nil(err)
		dMsg, err := cCodec.Decode(data)
		should.Nil(err)
		// 将原字节乱序
		rand.Shuffle(len(data), func(i, j int) {
			data[i], data[j] = data[j], data[i]
		})
		should.Equal(_getObjectValue(msg), _getObjectValue(dMsg))
	}
}
