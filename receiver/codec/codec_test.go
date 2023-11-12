package codec

import (
	"github.com/meow-pad/chinchilla/proto/receiver/pb"
	"github.com/stretchr/testify/require"
	"math/rand"
	"reflect"
	"testing"
)

func _getObjectValue(ptr any) any {
	ptrValue := reflect.ValueOf(ptr)
	return ptrValue.Elem().Interface()
}

func TestCodec_Message(t *testing.T) {
	should := require.New(t)
	msg := pb.HandshakeReq{
		RouterId: 1234,
		AuthKey:  "1234567",
		Service:  "test",
	}
	size := msg.XXX_Size()
	buf := make([]byte, size)
	buf1 := buf[:0]
	var err error
	buf1, err = msg.XXX_Marshal(buf1, false)
	should.Nil(err)
	msg1 := &pb.HandshakeReq{}
	err = msg1.XXX_Unmarshal(buf1)
	should.Nil(err)
}

func TestCodec_Req(t *testing.T) {
	should := require.New(t)
	handshakeReq := &HandshakeReq{
		pb.HandshakeReq{
			RouterId: 1234,
			AuthKey:  "1234567",
			Service:  "test",
		},
	}
	heartbeatReq := &HeartbeatReq{
		pb.HeartbeatReq{
			Payload: []byte{1, 2, 3, 4, 5},
		},
	}
	messageReq := &MessageReq{
		pb.MessageReq{
			Service: "test001",
			Payload: []byte{1, 2, 3, 4, 5},
		},
	}
	messages := []Message{handshakeReq, heartbeatReq, messageReq}
	cCodec := ClientCodec{}
	sCodec := ServerCodec{}
	for _, msg := range messages {
		data, err := cCodec.Encode(msg)
		should.Nil(err)
		dData, err := sCodec.Decode(data)
		should.Nil(err)
		// 为了保证最后比较相等能一致，这里也做一遍
		dMsg := dData.(Message)
		_, _ = dMsg.XXX_Marshal(make([]byte, 0, dMsg.XXX_Size()), false)
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
		pb.HandshakeRes{
			Code: 123,
		},
	}
	heartbeatRes := &HeartbeatRes{
		pb.HeartbeatRes{
			Code:    123,
			Payload: []byte{1, 2, 3, 4, 5},
		},
	}
	messageRes := &MessageRes{
		pb.MessageRes{
			Code:    123,
			Payload: []byte{1, 2, 3, 4, 5},
		},
	}
	messages := []any{handshakeRes, heartbeatRes, messageRes}
	cCodec := ClientCodec{}
	sCodec := ServerCodec{}
	for _, msg := range messages {
		data, err := sCodec.Encode(msg)
		should.Nil(err)
		dData, err := cCodec.Decode(data)
		should.Nil(err)
		// 为了保证最后比较相等能一致，这里也做一遍
		dMsg := dData.(Message)
		_, _ = dMsg.XXX_Marshal(make([]byte, 0, dMsg.XXX_Size()), false)
		// 将原字节乱序
		rand.Shuffle(len(data), func(i, j int) {
			data[i], data[j] = data[j], data[i]
		})
		should.Equal(_getObjectValue(msg), _getObjectValue(dMsg))
	}
}
