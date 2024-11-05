package trtest

import (
	"fmt"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"math"
	"time"
	"unsafe"
)

func newRemoteContext() (*RemoteContext, error) {
	remote := &RemoteContext{}
	// 将对象地址作为会话编号
	remote.Init(uint64(uintptr(unsafe.Pointer(remote))))
	if remote.Id() == SessionContextIdInvalid {
		return nil, fmt.Errorf("invalid remote context id, remote=%v", remote)
	}
	remote.SetDeadline(time.Now().UnixMilli() + withoutHandshakeTransferTimeoutMs)
	return remote, nil
}

type RemoteContext struct {
	session.BaseContext

	// 实例编号
	serverInstId string
}

func (ctx *RemoteContext) ShakeHand(serverInstId string) {
	ctx.serverInstId = serverInstId
	ctx.SetDeadline(math.MaxInt64)
}

func (ctx *RemoteContext) IsHandShook() bool {
	return len(ctx.serverInstId) > 0
}

func (ctx *RemoteContext) ServerInstId() string {
	return ctx.serverInstId
}

func NewLocalContext(serverInstId string) *LocalContext {
	local := &LocalContext{
		serverInstId: serverInstId,
	}
	local.Init(session.InvalidSessionId)
	local.SetDeadline(math.MaxInt64)
	return local
}

type LocalContext struct {
	session.BaseContext

	serverInstId string
}

func (ctx *LocalContext) SetSegmentFrame(buf []byte, amount uint16) {
}

func (ctx *LocalContext) GetSegmentFrame() ([]byte, uint16) {
	return nil, 0
}

func (ctx *LocalContext) ServerInstId() string {
	return ctx.serverInstId
}

func (ctx *LocalContext) IsHandShook() bool {
	return true
}
