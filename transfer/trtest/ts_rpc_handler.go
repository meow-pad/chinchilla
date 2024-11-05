package trtest

import (
	"fmt"
	"github.com/meow-pad/persian/utils/coding"
	"reflect"
)

type TsRPCHandler struct {
}

func (rpcHandler *TsRPCHandler) handleRequest(msg Message) (resp Message, err error) {
	defer coding.HandlePanicError("", func(aErr any) {
		if aErr != nil {
			err = fmt.Errorf("%v", aErr)
		}
	})
	switch tMsg := msg.(type) {
	case *RPCEchoReq:
		resp, err = rpcHandler.handleEchoReq(tMsg)
	default:
		err = fmt.Errorf("unknown message type: %T", reflect.TypeOf(msg).String())
	}
	return
}

func (rpcHandler *TsRPCHandler) handleEchoReq(msg *RPCEchoReq) (Message, error) {
	return &RPCEchoResp{
		Msg: msg.Msg,
	}, nil
}
