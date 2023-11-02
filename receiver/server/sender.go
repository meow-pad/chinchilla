package server

import (
	"github.com/meow-pad/chinchilla/receiver"
	"github.com/meow-pad/chinchilla/transfer/service"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"sync"
	"sync/atomic"
	"time"
)

func newSessionContext(server *receiver.Receiver, session session.Session) *SenderContext {
	ctx := &SenderContext{
		server:     server,
		session:    session,
		id:         session.Connection().Hash(),
		registered: false,
	}
	ctx.deadline.Store(time.Now().UnixMilli() + server.Options.UnregisteredSenderExpiration)
	return ctx
}

type SenderContext struct {
	server     *receiver.Receiver
	session    session.Session
	id         uint64
	deadline   atomic.Int64
	registered bool
	// 默认服务
	dfSrvName string
	dfService service.Service
	services  map[string]service.Service
	srvMu     sync.RWMutex
}

func (ctx *SenderContext) Id() uint64 {
	return ctx.id
}

func (ctx *SenderContext) Deadline() int64 {
	return ctx.deadline.Load()
}

func (ctx *SenderContext) UpdateDeadline() {
	if !ctx.registered {
		// 等待注册完成时间是固定的
		return
	}
	ctx.deadline.Store(time.Now().UnixMilli() + ctx.server.Options.RegisteredSenderExpiration)
}

func (ctx *SenderContext) SetRegistered(value bool) {
	if !value || ctx.registered {
		return
	}
	ctx.registered = true
	ctx.deadline.Store(time.Now().UnixMilli() + ctx.server.Options.RegisteredSenderExpiration)
}

func (ctx *SenderContext) IsRegistered() bool {
	return ctx.registered
}

func (ctx *SenderContext) SetService(srvName string, srv service.Service) {
	ctx.srvMu.Lock()
	defer ctx.srvMu.Unlock()
	if ctx.dfService == nil {
		ctx.dfSrvName = srvName
		ctx.dfService = srv
	} else {
		if ctx.services == nil {
			ctx.services = make(map[string]service.Service, 10)
		}
		ctx.services[srvName] = srv
	}
}

func (ctx *SenderContext) GetService(srvName string) service.Service {
	ctx.srvMu.RLock()
	defer ctx.srvMu.RUnlock()
	if ctx.dfSrvName == srvName {
		return ctx.dfService
	}
	if ctx.services != nil {
		return ctx.services[srvName]
	}
	return nil
}

func (ctx *SenderContext) GetDefaultService() (string, service.Service) {
	ctx.srvMu.RLock()
	defer ctx.srvMu.RUnlock()
	return ctx.dfSrvName, ctx.dfService
}
