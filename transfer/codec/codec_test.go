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
	segmentMsg := &SegmentMsg{
		Amount: 12345,
		Seq:    123,
		Frame:  []byte{1, 2, 3, 4, 5},
	}
	handshakeReq := &HandshakeReq{
		Id:        "1234",
		AuthKey:   "12345",
		Service:   "test",
		ServiceId: "123",
		ConnIds:   []uint64{123, 456},
		RouterIds: []string{"123", "456"},
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
	srvInstIRes := &ServiceInstIRes{
		ServiceName:    "123",
		ServiceInstArr: []string{"123", "456"},
	}
	messages := []any{segmentMsg, handshakeReq, registerSReq, unregisterReq, heartbeatSReq, messageSReq, srvInstIRes}
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
	segmentMsg := &SegmentMsg{
		Amount: 12345,
		Seq:    123,
		Frame:  []byte{1, 2, 3, 4, 5},
	}
	handshakeRes := &HandshakeRes{
		Code: 123,
	}
	registerSRes := &RegisterSRes{
		ConnId:   12345,
		Code:     123,
		RouterId: "12345",
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
	broadcastSRes := &BroadcastSRes{
		ConnIds: []uint64{123, 456, 789},
		Payload: []byte{1, 2, 3, 4, 5, 6},
	}
	messageRouter := &MessageRouter{
		RouterService: "456",
		RouterType:    -123,
		RouterId:      "abc",
		Payload:       []byte{1, 2, 3, 4, 5},
	}
	serviceInstIReq := &ServiceInstIReq{
		ServiceName: "654",
	}
	messages := []any{segmentMsg, handshakeRes, registerSRes, unregisterSRes,
		heartbeatSRes, messageSRes, broadcastSRes, messageRouter, serviceInstIReq}
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
