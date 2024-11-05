package trtest

import (
	"context"
	"github.com/meow-pad/chinchilla/transfer/codec"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"github.com/meow-pad/persian/utils/collections"
	"github.com/meow-pad/persian/utils/timewheel"
	"sync/atomic"
	"time"
)

const (
	tsUserMMgrTickInterval = 5 * time.Second
	userSessionTimeoutMSec = 60 * 1000
)

type UserSession struct {
	connId         uint64
	sess           session.Session
	lastAccessTime int64
	uid            atomic.Int32
}

func (sess *UserSession) SendMessage(msg Message) {
	msgBytes, err := msgCodec.Encode(msg)
	if err != nil {
		plog.Error("encode message failed",
			pfield.Uint64("connId", sess.connId), pfield.Error(err))
		return
	}
	sess.sess.SendMessage(&codec.MessageSRes{
		ConnId:  sess.connId,
		Payload: msgBytes,
	})
}

func (sess *UserSession) SendGatewayMessage(msg any) {
	sess.sess.SendMessage(msg)
}

func (sess *UserSession) Access() {
	sess.lastAccessTime = time.Now().UnixMilli()
}

func newTSUserManager(runtime *TransferRuntime) *TSUserManager {
	return &TSUserManager{Runtime: runtime}
}

type TSUserManager struct {
	Runtime *TransferRuntime

	userSessions collections.SyncMap[uint64, *UserSession]
	users        collections.SyncMap[int32, *UserSession]
	tickTask     *timewheel.Task
}

func (handler *TSUserManager) Start() error {
	handler.tickTask = handler.Runtime.Timer.AddCron(tsUserMMgrTickInterval, handler.tick)
	return nil
}

func (handler *TSUserManager) Stop(ctx context.Context) error {
	if handler.tickTask != nil {
		if err := handler.Runtime.Timer.Remove(handler.tickTask); err != nil {
			return err
		}
	}
	return nil
}

func (handler *TSUserManager) tick() {
	handler.userSessions.Range(func(key uint64, value *UserSession) bool {
		if time.Now().UnixMilli()-value.lastAccessTime > 30*1000 {
			handler.RemoveUserSession(key)
		}
		return true
	})
}

func (handler *TSUserManager) GetUserSession(connId uint64) *UserSession {
	sess, _ := handler.userSessions.Load(connId)
	return sess
}

func (handler *TSUserManager) SetUserSession(connId uint64, sess *UserSession) {
	sess.Access()
	handler.userSessions.Store(connId, sess)
}

func (handler *TSUserManager) RemoveUserSession(connId uint64) {
	uSess, _ := handler.userSessions.Delete(connId)
	if uSess == nil {
		return
	}
	if uSess.uid.Load() != UIDInvalid {
		handler.RemoveUser(uSess.uid.Load())
	}
}

func (handler *TSUserManager) GetUser(uid int32) *UserSession {
	uSess, _ := handler.users.Load(uid)
	return uSess
}

func (handler *TSUserManager) AddUser(uid int32, sess *UserSession) {
	eSess, _ := handler.userSessions.Load(sess.connId)
	if eSess == nil {
		handler.SetUserSession(sess.connId, sess)
	}
	oldSess, _ := handler.users.Load(uid)
	if oldSess != nil {
		if oldSess.connId != sess.connId {
			// 顶号，旧连接的登出
			sess.SendGatewayMessage(&codec.RegisterSRes{
				ConnId: sess.connId,
				Code:   UserCodeOtherDeviceLogin,
			})
			handler.RemoveUserSession(oldSess.connId)
		} else {
			// same session
			sess.SendGatewayMessage(&codec.RegisterSRes{
				ConnId: sess.connId,
			})
			return
		}
	}
	sess.uid.Store(uid)
	handler.users.Store(uid, sess)
	sess.SendGatewayMessage(&codec.RegisterSRes{
		ConnId: sess.connId,
	})
}

func (handler *TSUserManager) RemoveUser(uid int32) {
	uSess, _ := handler.users.Delete(uid)
	if uSess != nil {
		handler.userSessions.Delete(uSess.connId)
	}
}
