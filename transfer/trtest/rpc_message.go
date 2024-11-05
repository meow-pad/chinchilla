package trtest

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

type RPCCoder struct {
}

func (coder *RPCCoder) Encode(msg Message) ([]byte, error) {
	dataBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	output := make([]byte, 2+len(dataBytes))
	binary.LittleEndian.PutUint16(output, msg.Type())
	copy(output[2:], dataBytes)
	return output, nil
}

func (coder *RPCCoder) Decode(input []byte) (Message, error) {
	typeBytes := input[:2]
	msgType := binary.LittleEndian.Uint16(typeBytes)
	msg, err := newRPCMessage(msgType)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(input[2:], msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

var (
	rpcMsgCodec RPCCoder
)

const (
	RPCMessageTypeErrRes  = 2
	RPCMessageTypeEchoReq = 3
	RPCMessageTypeEchoRes = 4
)

type RPCErrResp struct {
	Err string
}

func (msg *RPCErrResp) Type() uint16 {
	return RPCMessageTypeErrRes
}

type RPCEchoReq struct {
	Msg string
}

func (msg *RPCEchoReq) Type() uint16 {
	return RPCMessageTypeEchoReq
}

type RPCEchoResp struct {
	Msg string
}

func (msg *RPCEchoResp) Type() uint16 {
	return RPCMessageTypeEchoRes
}

func newRPCMessage(msgType uint16) (Message, error) {
	switch msgType {
	case RPCMessageTypeErrRes:
		return &RPCErrResp{}, nil
	case RPCMessageTypeEchoReq:
		return &RPCEchoReq{}, nil
	case RPCMessageTypeEchoRes:
		return &RPCEchoResp{}, nil
	default:
		return nil, fmt.Errorf("unknown rpc message type: %d", msgType)
	}
}
