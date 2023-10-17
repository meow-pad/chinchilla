package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/meow-pad/chinchilla/transfer/codec"
	"github.com/meow-pad/persian/errdef"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pnet/tcp/client"
	"github.com/meow-pad/persian/utils/json"
	"math"
	"sync/atomic"
	"time"
)

const (
	StateInitialized = iota
	StateConnecting  // 连接中
	StateConnected   // 已连接
	StateDisabled    // 实例不可用
	StateStopped     // 实例关闭
)

var (
	ErrConnectClientFirst   = errors.New("connect client first")
	ErrConnectingClient     = errors.New("client is connecting")
	ErrNotConnected         = errors.New("not in connected state")
	ErrFrequentReconnection = errors.New("reconnection is too frequent")
	ErrNoCertification      = errors.New("obtain certification first")
)

var (
	reconnectInterval = []int64{1000, 2000, 4000, 8000, 10_000}
)

func newRemoteService(manager *Manager, srvInfo Info) (*Remote, error) {
	remote := &Remote{}
	if err := remote.init(manager, srvInfo); err != nil {
		return nil, err
	}
	return remote, nil
}

// Remote 远程服务实例
type Remote struct {
	manager    *Manager
	inner      *client.Client
	info       Info
	state      atomic.Int32
	certified  atomic.Bool
	dialCtx    context.Context
	dialCancel context.CancelFunc
	// 最近连接时间，以便使用退避算法
	lastConnect  atomic.Int64
	reconnectLvl int
	// 最终关闭时间
	deadline int64
}

func (remoteSrv *Remote) init(manager *Manager, srvInfo Info) error {
	if manager == nil {
		return errdef.ErrInvalidParams
	}
	remoteSrv.manager = manager
	remoteSrv.info = srvInfo
	remoteSrv.state.Store(StateInitialized)
	remoteSrv.deadline = math.MaxInt64
	return nil
}

func (remoteSrv *Remote) UpdateInfo(srvInst Info) error {
	if remoteSrv.state.Load() == StateStopped {
		return ErrStoppedInstance
	}
	if remoteSrv.info.InstanceId != srvInst.InstanceId || remoteSrv.info.Ip != srvInst.Ip || remoteSrv.info.Port != srvInst.Port {
		plog.Warn("invalid service instance:",
			pfield.String("old", json.ToString(remoteSrv.info)),
			pfield.String("new", json.ToString(remoteSrv.info)))
		return errdef.ErrInvalidParams
	}
	oldOne := remoteSrv.info
	remoteSrv.info = srvInst
	// 值改变则判定
	if remoteSrv.info.Healthy != oldOne.Healthy || remoteSrv.info.Enable != oldOne.Enable {
		if !remoteSrv.info.Enable {
			remoteSrv.state.Store(StateDisabled)
			remoteSrv.deadline = time.Now().UnixMilli() + remoteSrv.manager.transfer.Gateway.Options.TransferClientDisableTimeout
		} else {
			if !remoteSrv.info.Healthy {
				remoteSrv.state.Store(StateDisabled)
				remoteSrv.deadline = math.MaxInt64
			} else {
				if remoteSrv.state.Load() == StateDisabled {
					remoteSrv.state.Store(StateInitialized)
					remoteSrv.deadline = math.MaxInt64
					// 尝试连接
					if err := remoteSrv.connect(); err != nil {
						plog.Error("connect error:", pfield.Error(err))
					}
				} // end of if
			} // end of else
		} // end of else
	} // end of if
	return nil
}

func (remoteSrv *Remote) Info() Info {
	return remoteSrv.info
}

// disable
//
//	@Description: 设为关闭状态
//	@receiver sClient
//	@return error
func (remoteSrv *Remote) disable() error {
	state := remoteSrv.state.Load()
	switch state {
	case StateStopped:
		return ErrStoppedInstance
	default:
		remoteSrv.info.Enable = false
		if state != StateDisabled || remoteSrv.deadline == math.MaxInt64 {
			remoteSrv.deadline = time.Now().UnixMilli() + remoteSrv.manager.transfer.Gateway.Options.TransferClientDisableTimeout
		}
		remoteSrv.state.Store(StateDisabled)
	}
	return nil
}

// getReconnectInterval
//
//	@Description:  重连时间间隔
//	@receiver sClient
//	@return int64
func (remoteSrv *Remote) getReconnectInterval() int64 {
	return reconnectInterval[remoteSrv.reconnectLvl%len(reconnectInterval)]
}

// connect
//
//	@Description: 连接
//	@receiver sClient
//	@return error
func (remoteSrv *Remote) connect() error {
	// 触发可见性
	state := remoteSrv.state.Load()
	switch state {
	case StateDisabled:
		return ErrDisabledService
	case StateStopped:
		return ErrStoppedInstance
	default:
	}
	if remoteSrv.dialCtx != nil {
		// 如果上次连接还没结束则返回
		return ErrConnectingClient
	}
	// 连接间隔
	now := time.Now().UnixMilli()
	if (remoteSrv.lastConnect.Load() + remoteSrv.getReconnectInterval()) >= now {
		return ErrFrequentReconnection
	} else {
		remoteSrv.lastConnect.Store(now)
		remoteSrv.reconnectLvl++
	}
	// 创建客户端
	options := remoteSrv.manager.transfer.Gateway.Options
	clientOpts := []client.Option{
		client.WithReadBufferCap(options.TransferClientReadBufferCap),
		client.WithWriteBufferCap(options.TransferClientWriteBufferCap),
		client.WithWriteQueueCap(options.TransferClientWriteQueueCap),
		client.WithTCPKeepAlive(options.TransferClientTCPKeepAlive),
		client.WithSocketRecvBuffer(options.TransferClientSocketRecvBuffer),
		client.WithSocketSendBuffer(options.TransferClientSocketSendBuffer),
	}
	cCodec, err := codec.NewMessageCodec(remoteSrv.manager.clientCodec)
	if err != nil {
		return err
	}
	tClient, err := client.NewClient(cCodec, &Listener{client: remoteSrv}, clientOpts...)
	if err == nil {
		return err
	}
	remoteSrv.dialCtx, remoteSrv.dialCancel = context.WithTimeout(context.Background(), options.TransferClientDialTimeout)
	if !remoteSrv.state.CompareAndSwap(state, StateConnecting) {
		return errors.New("invalid state")
	}
	go remoteSrv._connect(tClient)
	return nil
}

// _connect
//
//	@Description: 真实连接操作
//	@receiver sClient
//	@param tClient
func (remoteSrv *Remote) _connect(tClient *client.Client) {
	err := tClient.Dial(remoteSrv.dialCtx, fmt.Sprintf("%s:%d", remoteSrv.info.Ip, remoteSrv.info.Port))
	if err != nil {
		plog.Error("client connect error:", pfield.Error(err))
	}
	remoteSrv.inner = tClient
	// 重置重连间隔
	remoteSrv.reconnectLvl = 0
	if remoteSrv.dialCancel != nil {
		remoteSrv.dialCancel()
		remoteSrv.dialCtx, remoteSrv.dialCancel = nil, nil
	}
	if !remoteSrv.state.CompareAndSwap(StateConnecting, StateConnected) {
		plog.Error("cant change state to StateConnected", pfield.Int32("curState", remoteSrv.state.Load()))
	}
}

// handshake
//
//	@Description: 握手
//	@receiver sClient
func (remoteSrv *Remote) handshake() {
	if remoteSrv.certified.Load() {
		return
	}
	gateway := remoteSrv.manager.transfer.Gateway
	options := gateway.Options
	// 发送握手
	remoteSrv.inner.SendMessage(&codec.HandshakeReq{
		Id:        gateway.AppInfo.Id(), // 当前服务Id
		AuthKey:   options.TransferServiceAuthKey,
		Service:   remoteSrv.info.ServiceName, // 对方服务名
		ServiceId: remoteSrv.info.InstanceId,  // 对方实例Id
	})
}

func (remoteSrv *Remote) onHandshake() {
	remoteSrv.certified.CompareAndSwap(false, true)
	plog.Debug("on handshake", pfield.Any("info", remoteSrv.info))
}

// KeepAlive
//
//	@Description: 连接保活
//	@receiver sClient
//	@return 存活状态
func (remoteSrv *Remote) KeepAlive() bool {
	switch remoteSrv.state.Load() {
	case StateDisabled:
		// 判定过期的disable则关闭，已经关闭的则移除
		if time.Now().UnixMilli() > remoteSrv.deadline {
			if remoteSrv.state.CompareAndSwap(StateDisabled, StateStopped) {
				return false
			} else {
				// ???状态改变了
				return true
			}
		}
		return true
	case StateStopped:
		return false
	default:
		if remoteSrv.inner.IsClosed() {
			err := remoteSrv.connect()
			if err != nil {
				plog.Error("reconnect error:", pfield.Error(err))
			}
		} else {
			if !remoteSrv.certified.Load() {
				remoteSrv.handshake()
			} else {
				// 连接正常则发送心跳
				remoteSrv.inner.SendMessage(codec.HeartbeatSReq{})
			}
		}
	} // end of switch
	return true
}

// checkAlive
//
//	@Description: 检查连接是否保持
//	@receiver sClient
//	@return error
func (remoteSrv *Remote) checkAlive() error {
	switch remoteSrv.state.Load() {
	case StateInitialized:
		return ErrConnectClientFirst
	case StateConnecting:
		return ErrConnectingClient
	case StateDisabled:
		return ErrDisabledService
	case StateStopped:
		return ErrStoppedInstance
	default:
	}
	if remoteSrv.inner.IsClosed() {
		err := remoteSrv.connect()
		if err != nil {
			return err
		} else {
			return ErrConnectingClient
		}
	} else {
		if !remoteSrv.certified.Load() {
			return ErrNoCertification
		}
	}
	return nil
}

func (remoteSrv *Remote) SendMessage(msg any) error {
	if err := remoteSrv.checkAlive(); err != nil {
		return err
	}
	remoteSrv.inner.SendMessage(msg)
	return nil
}

func (remoteSrv *Remote) closeConn() error {
	if remoteSrv.inner != nil {
		return remoteSrv.inner.Close()
	}
	return nil
}

func (remoteSrv *Remote) IsConnClosed() bool {
	if remoteSrv.inner != nil {
		return remoteSrv.inner.IsClosed()
	}
	return false
}

func (remoteSrv *Remote) IsEnable() bool {
	switch remoteSrv.state.Load() {
	case StateDisabled, StateStopped:
		return false
	default:
		return true
	}
}

func (remoteSrv *Remote) IsStopped() bool {
	return remoteSrv.state.Load() == StateStopped
}

func (remoteSrv *Remote) Stop(ctx context.Context) error {
	state := remoteSrv.state.Load()
	if state == StateStopped {
		return ErrStoppedInstance
	}
	if remoteSrv.dialCancel != nil {
		remoteSrv.dialCancel()
		remoteSrv.dialCtx, remoteSrv.dialCancel = nil, nil
	}
	if remoteSrv.inner != nil {
		if err := remoteSrv.inner.CloseWithContext(ctx); err != nil {
			plog.Error("close connection error:", pfield.Error(err))
		}
	}
	return nil
}
