package codec

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/meow-pad/chinchilla/proto/receiver/pb"
	"strconv"
)

const (
	TypeHandshake uint8 = iota + 1
	TypeHeartbeat
	TypeMessage

	maxStringLen  = 1<<16 - 1
	maxServiceLen = 1<<8 - 1
)

type Message interface {
	proto.Message

	XXX_Unmarshal(b []byte) error
	XXX_Marshal(b []byte, deterministic bool) ([]byte, error)
	XXX_Size() int

	Type() uint8
}

type HandshakeReq struct {
	pb.HandshakeReq

	routerId uint64
}

func (req *HandshakeReq) RouterId() uint64 {
	return req.routerId
}

func (req *HandshakeReq) InitRouterId() error {
	if len(req.HandshakeReq.RouterId) > 0 {
		rId, err := strconv.ParseUint(req.HandshakeReq.RouterId, 10, 64)
		if err != nil {
			return err
		}
		req.routerId = rId
	}
	return nil
}

func (req *HandshakeReq) Type() uint8 {
	return TypeHandshake
}

type HandshakeRes struct {
	pb.HandshakeRes
}

func (res *HandshakeRes) Type() uint8 {
	return TypeHandshake
}

type HeartbeatReq struct {
	pb.HeartbeatReq
}

func (req *HeartbeatReq) Type() uint8 {
	return TypeHeartbeat
}

type HeartbeatRes struct {
	pb.HeartbeatRes
}

func (res *HeartbeatRes) Type() uint8 {
	return TypeHeartbeat
}

type MessageReq struct {
	pb.MessageReq
}

func (req *MessageReq) Type() uint8 {
	return TypeMessage
}

type MessageRes struct {
	pb.MessageRes
}

func (res *MessageRes) Type() uint8 {
	return TypeMessage
}

func newReqMessage(msgType uint8) (Message, error) {
	switch msgType {
	case TypeMessage:
		return new(MessageReq), nil
	case TypeHeartbeat:
		return new(HeartbeatReq), nil
	case TypeHandshake:
		return new(HandshakeReq), nil
	default:
		return nil, fmt.Errorf("unknown message type:%d", msgType)
	}
}

func newResMessage(msgType uint8) (Message, error) {
	switch msgType {
	case TypeMessage:
		return new(MessageRes), nil
	case TypeHeartbeat:
		return new(HeartbeatRes), nil
	case TypeHandshake:
		return new(HandshakeRes), nil
	default:
		return nil, fmt.Errorf("unknown message type:%d", msgType)
	}
}
