package trtest

import (
	"fmt"
	"github.com/meow-pad/persian/utils/coding"
)

type TsMsgHandler struct {
}

func (msgHandler *TsMsgHandler) handle(uSess *UserSession, msg Message) (err error) {
	defer coding.HandlePanicError("", func(aErr any) {
		err = fmt.Errorf("%v", aErr)
	})
	switch tMsg := msg.(type) {
	case *EchoReq:
		return msgHandler.handleEchoReq(uSess, tMsg)
	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
}

func (msgHandler *TsMsgHandler) handleEchoReq(uSess *UserSession, msg *EchoReq) error {
	uSess.SendMessage(&EchoResp{
		ReqId: msg.ReqId,
		Msg:   msg.Msg,
	})
	return nil
}
