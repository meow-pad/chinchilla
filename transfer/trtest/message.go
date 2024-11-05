package trtest

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

type Message interface {
	Type() uint16
}

type Coder struct {
}

func (coder *Coder) Encode(msg Message) ([]byte, error) {
	dataBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	output := make([]byte, 2+len(dataBytes))
	binary.LittleEndian.PutUint16(output, msg.Type())
	copy(output[2:], dataBytes)
	return output, nil
}

func (coder *Coder) Decode(input []byte) (Message, error) {
	typeBytes := input[:2]
	msgType := binary.LittleEndian.Uint16(typeBytes)
	msg, err := newMessage(msgType)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(input[2:], msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

const (
	MessageTypeLoginReq uint16 = iota + 1
	MessageTypeLoginResp
	MessageTypeEchoReq
	MessageTypeEchoResp
	MessageTypeHeartbeatReq
	MessageTypeHeartbeatResp
)

var (
	msgCodec Coder
)

type LoginReq struct {
	ReqId int64
	Uid   int32
}

func (req *LoginReq) Type() uint16 {
	return MessageTypeLoginReq
}

type LoginResp struct {
	ReqId int64
	Msg   string
}

func (resp *LoginResp) Type() uint16 {
	return MessageTypeLoginResp
}

type EchoReq struct {
	ReqId int64
	Msg   string
}

func (req *EchoReq) Type() uint16 {
	return MessageTypeEchoReq
}

type EchoResp struct {
	ReqId int64
	Msg   string
}

func (resp *EchoResp) Type() uint16 {
	return MessageTypeEchoResp
}

type HeartbeatReq struct {
	ReqId int64
	CTime int64
}

func (req *HeartbeatReq) Type() uint16 {
	return MessageTypeHeartbeatReq
}

type HeartbeatResp struct {
	ReqId int64
	CTime int64
	STime int64
}

func (req *HeartbeatResp) Type() uint16 {
	return MessageTypeHeartbeatResp
}

func newMessage(msgType uint16) (Message, error) {
	switch msgType {
	case MessageTypeLoginReq:
		return &LoginReq{}, nil
	case MessageTypeLoginResp:
		return &LoginResp{}, nil
	case MessageTypeEchoReq:
		return &EchoReq{}, nil
	case MessageTypeEchoResp:
		return &EchoResp{}, nil
	case MessageTypeHeartbeatReq:
		return &HeartbeatReq{}, nil
	case MessageTypeHeartbeatResp:
		return &HeartbeatResp{}, nil
	default:
		return nil, fmt.Errorf("unknown message type: %d", msgType)
	}
}
