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
		Id:        "1234",
		AuthKey:   "12345",
		Service:   "test",
		ServiceId: "123",
	}
	registerSReq := &RegisterSReq{
		ConnId:  12345,
		Payload: []byte{1, 2, 3, 4, 5},
	}
	unregisterReq := &UnregisterSReq{
		ConnId: 12345,
	}
	heartbeatSReq := &HeartbeatSReq{
		ConnId:  12345,
		Payload: []byte{1, 2, 3, 4, 5},
	}
	messageSReq := &MessageSReq{
		ConnId:  12345,
		Payload: []byte{1, 2, 3, 4, 5},
	}
	messages := []any{handshakeReq, registerSReq, unregisterReq, heartbeatSReq, messageSReq}
	cCodec := ClientCodec{byteOrder: binary.BigEndian}
	sCodec := ServerCodec{byteOrder: binary.BigEndian}
	for _, msg := range messages {
		data, err := cCodec.Encode(msg)
		should.Nil(err)
		dMsg, err := sCodec.Decode(data)
		should.Nil(err)
		// 乱序，确保有正确拷贝
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
	registerSRes := &RegisterSRes{
		ConnId:   12345,
		Code:     123,
		RouterId: 12345,
		Payload:  []byte{1, 2, 3, 4, 5},
	}
	unregisterSRes := &UnregisterSRes{
		ConnId: 12345,
	}
	heartbeatSRes := &HeartbeatSRes{
		ConnId:  12345,
		Payload: []byte{1, 2, 3, 4, 5},
	}
	messageSRes := &MessageSRes{
		ConnId:  12345,
		Payload: []byte{1, 2, 3, 4, 5},
	}
	messages := []any{handshakeRes, registerSRes, unregisterSRes, heartbeatSRes, messageSRes}
	cCodec := ClientCodec{byteOrder: binary.BigEndian}
	sCodec := ServerCodec{byteOrder: binary.BigEndian}
	for _, msg := range messages {
		data, err := sCodec.Encode(msg)
		should.Nil(err)
		dMsg, err := cCodec.Decode(data)
		should.Nil(err)
		// 乱序，确保有正确拷贝
		rand.Shuffle(len(data), func(i, j int) {
			data[i], data[j] = data[j], data[i]
		})
		should.Equal(_getObjectValue(msg), _getObjectValue(dMsg))
	}
}
