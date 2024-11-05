package trtest

import (
	"context"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"github.com/meow-pad/persian/utils/collections"
	"github.com/meow-pad/persian/utils/timewheel"
	"time"
)

const (
	sessionManagerCheckInterval = 30 * time.Second
)

func newTSSessManager(runtime *TransferRuntime) *TSSessManager {
	return &TSSessManager{
		Runtime: runtime,
	}
}

type TSSessManager struct {
	Runtime *TransferRuntime

	sessMap  collections.SyncMap[string, session.Session]
	tickTask *timewheel.Task
}

func (sessMgr *TSSessManager) AddSess(sessServiceId string, sess session.Session) {
	sessMgr.sessMap.Store(sessServiceId, sess)
}

func (sessMgr *TSSessManager) RemoveSess(sessServiceId string) {
	sessMgr.sessMap.Delete(sessServiceId)
}

func (sessMgr *TSSessManager) GetSession(serviceId string) session.Session {
	sess, _ := sessMgr.sessMap.Load(serviceId)
	return sess
}

func (sessMgr *TSSessManager) GetOneSession() session.Session {
	var sess session.Session
	sessMgr.sessMap.Range(func(key string, value session.Session) bool {
		sess = value
		return false
	})
	return sess
}

func (sessMgr *TSSessManager) Start() error {
	sessMgr.tickTask = sessMgr.Runtime.Timer.AddCron(sessionManagerCheckInterval, sessMgr.tick)
	return nil
}

func (sessMgr *TSSessManager) Stop(ctx context.Context) error {
	if err := sessMgr.Runtime.Timer.Remove(sessMgr.tickTask); err != nil {
		plog.Error("failed to remove tick task", pfield.Error(err))
	}
	return nil
}

func (sessMgr *TSSessManager) tick() {
	sessMgr.sessMap.Range(func(key string, value session.Session) bool {
		if value.IsClosed() {
			sessMgr.sessMap.Delete(key)
		}
		return true
	})
}
