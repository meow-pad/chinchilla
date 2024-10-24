package transfer

import (
	"context"
	"errors"
	"fmt"
	"github.com/meow-pad/chinchilla/transfer/codec"
	"github.com/meow-pad/chinchilla/transfer/common"
	"github.com/meow-pad/chinchilla/transfer/service"
	"github.com/meow-pad/persian/errdef"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pnet/tcp/client"
	tcodec "github.com/meow-pad/persian/frame/pnet/tcp/codec"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"github.com/meow-pad/persian/utils/json"
	"math"
	"sync"
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
	reconnectInterval = []int64{2000, 2000, 4000, 8000, 10_000}
)

func NewRemoteService(manager *Manager, srvInfo common.Info) (*Remote, error) {
	remote := &Remote{}
	if err := remote.init(manager, srvInfo); err != nil {
		return nil, err
	}
	return remote, nil
}

func newConnectContext(onConnect func() error,
	onConnected func(tClient *client.Client, err error), onCancelConnect func()) *connectContext {
	return &connectContext{
		onConnect:       onConnect,
		onConnected:     onConnected,
		onCancelConnect: onCancelConnect,
	}
}

type connectContext struct {
	dialLock   sync.Mutex
	dialCtx    context.Context
	dialCancel context.CancelFunc
	// 最近连接时间，以便使用退避算法
	lastConnect  atomic.Int64
	reconnectLvl atomic.Int32
	// 连接触发函数
	onConnect func() error
	// 连接完成函数
	onConnected func(tClient *client.Client, err error)
	// 取消连接
	onCancelConnect func()
}

// getReconnectInterval
//
//	@Description:  重连时间间隔
//	@receiver sClient
//	@return int64
func (connCtx *connectContext) getReconnectInterval() int64 {
	return reconnectInterval[int(connCtx.reconnectLvl.Load())%len(reconnectInterval)]
}

// canConnect
//
//	@Description: 是否能进行连接
//	@receiver connCtx
//	@return error 错误信息
func (connCtx *connectContext) canConnect() error {
	now := time.Now().UnixMilli()
	// 连接退避时间间隔
	if (connCtx.lastConnect.Load() + connCtx.getReconnectInterval()) >= now {
		return ErrFrequentReconnection
	}
	return nil
}

// beforeConnect
//
//	@Description: 连接前准备操作
func (connCtx *connectContext) beforeConnect() (error, context.Context) {
	now := time.Now().UnixMilli()
	connCtx.dialLock.Lock()
	defer connCtx.dialLock.Unlock()
	if connCtx.dialCtx != nil {
		// 之前连接的状态还存在则清理
		connCtx.dialCancel()
		connCtx.dialCtx = nil
		connCtx.dialCancel = nil
	}
	connCtx.lastConnect.Store(now)
	connCtx.reconnectLvl.Add(1)
	// 获取新的连接状态
	connCtx.dialCtx, connCtx.dialCancel = context.WithCancel(context.Background())
	// 触发连接时函数
	if connCtx.onConnect != nil {
		if err := connCtx.onConnect(); err != nil {
			connCtx.dialCancel()
			connCtx.dialCtx = nil
			connCtx.dialCancel = nil
			return err, nil
		}
	}
	return nil, connCtx.dialCtx
}

// afterConnect
//
//	@Description: 连接后续操作
//	@receiver connCtx
//	@param err
func (connCtx *connectContext) afterConnect(tClient *client.Client, err error) {
	connCtx.dialLock.Lock()
	defer connCtx.dialLock.Unlock()
	if connCtx.dialCtx != nil {
		connCtx.dialCtx = nil
		connCtx.dialCancel = nil
	}
	if err == nil {
		connCtx.reconnectLvl.Store(0)
	}
	if connCtx.onConnected != nil {
		connCtx.onConnected(tClient, err)
	}
}

// cancelConnect
//
//	@Description: 取消当前的连接
//	@receiver connCtx
func (connCtx *connectContext) cancelConnect() {
	connCtx.dialLock.Lock()
	defer connCtx.dialLock.Unlock()
	if connCtx.dialCancel != nil {
		connCtx.dialCancel()
		connCtx.dialCtx, connCtx.dialCancel = nil, nil
		if connCtx.onCancelConnect != nil {
			connCtx.onCancelConnect()
		}
	}
}

// Remote 远程服务实例
type Remote struct {
	manager   *Manager
	codec     tcodec.Codec
	inner     *client.Client
	info      common.Info
	state     atomic.Int32
	certified atomic.Bool
	// 连接上下文
	connectCtx *connectContext
	// 最终关闭时间
	deadline int64
	// 分段数据
	segmentFrameBuf    []byte
	segmentFrameAmount uint16
}

func (remoteSrv *Remote) init(manager *Manager, srvInfo common.Info) error {
	if manager == nil {
		return errdef.ErrInvalidParams
	}
	remoteSrv.manager = manager
	remoteSrv.info = srvInfo
	remoteSrv.state.Store(StateInitialized)
	remoteSrv.deadline = math.MaxInt64
	// 编码器
	options := remoteSrv.manager.transfer.Options
	cCodec, err := codec.NewCodec(remoteSrv.manager.clientCodec, options.TransferMessageWarningSize)
	if err != nil {
		return err
	}
	remoteSrv.codec = cCodec
	remoteSrv.connectCtx = newConnectContext(remoteSrv.onConnect, remoteSrv.onConnected, remoteSrv.onCancelConnect)
	return nil
}

func (remoteSrv *Remote) UpdateInfo(srvInst common.Info) error {
	if remoteSrv.state.Load() == StateStopped {
		return service.ErrStoppedInstance
	}
	if remoteSrv.info.ServiceId() != srvInst.ServiceId() || remoteSrv.info.Ip != srvInst.Ip || remoteSrv.info.Port != srvInst.Port {
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
			remoteSrv.deadline = time.Now().UnixMilli() + remoteSrv.manager.transfer.Options.TransferClientDisableTimeout
			plog.Debug("disable transfer client:", pfield.String("serviceId", remoteSrv.info.ServiceId()))
		} else {
			if !remoteSrv.info.Healthy {
				remoteSrv.state.Store(StateDisabled)
				remoteSrv.deadline = math.MaxInt64
			} else {
				if remoteSrv.state.Load() == StateDisabled {
					remoteSrv.state.Store(StateInitialized)
					remoteSrv.deadline = math.MaxInt64
					// 尝试连接
					if err := remoteSrv.Connect(); err != nil {
						plog.Error("transfer client connect error:", pfield.Error(err))
					}
				} // end of if
			} // end of else
		} // end of else
	} // end of if
	return nil
}

func (remoteSrv *Remote) Info() common.Info {
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
		return service.ErrStoppedInstance
	default:
		remoteSrv.info.Enable = false
		if state != StateDisabled || remoteSrv.deadline == math.MaxInt64 {
			remoteSrv.deadline = time.Now().UnixMilli() +
				remoteSrv.manager.transfer.Options.TransferClientDisableTimeout
		}
		remoteSrv.state.Store(StateDisabled)
	}
	return nil
}

func (remoteSrv *Remote) canStateConnecting(state int32) error {
	switch state {
	case StateDisabled:
		return service.ErrDisabledService
	case StateStopped:
		return service.ErrStoppedInstance
	default:
		return nil
	}
}

func (remoteSrv *Remote) onConnect() error {
	state := remoteSrv.state.Load()
	if err := remoteSrv.canStateConnecting(state); err != nil {
		return err
	}
	if !remoteSrv.state.CompareAndSwap(state, StateConnecting) {
		return fmt.Errorf("(transfer client) invalid state:%d", state)
	}
	return nil
}

func (remoteSrv *Remote) onConnected(tClient *client.Client, err error) {
	cutState := remoteSrv.state.Load()
	if cutState != StateConnecting {
		plog.Error("transfer client state is not in connecting on connected",
			pfield.Int32("curState", cutState))
		return
	}
	if err != nil {
		// 回滚状态
		if !remoteSrv.state.CompareAndSwap(StateConnecting, StateInitialized) {
			plog.Error("transfer client cant change state to StateInitialized on connecting",
				pfield.Int32("curState", cutState))
		}
	} else {
		remoteSrv.inner = tClient
		// 进入链接状态
		if !remoteSrv.state.CompareAndSwap(StateConnecting, StateConnected) {
			plog.Error("transfer client cant change state to StateConnected on connecting",
				pfield.Int32("curState", remoteSrv.state.Load()))
		}
	}
}

func (remoteSrv *Remote) onCancelConnect() {
	cutState := remoteSrv.state.Load()
	if cutState != StateConnecting {
		plog.Error("transfer client state is not in connecting on cancel connecting",
			pfield.Int32("curState", cutState))
		return
	}
	if !remoteSrv.state.CompareAndSwap(StateConnecting, StateInitialized) {
		plog.Error("transfer client cant change state to StateInitialized on cancel connecting",
			pfield.Int32("curState", cutState))
	}
}

// Connect
//
//	@Description: 连接
//	@receiver sClient
//	@return error
func (remoteSrv *Remote) Connect() error {
	// 触发可见性
	state := remoteSrv.state.Load()
	if err := remoteSrv.canStateConnecting(state); err != nil {
		return err
	}

	if err := remoteSrv.connectCtx.canConnect(); err != nil {
		return err
	}

	// 创建客户端
	options := remoteSrv.manager.transfer.Options
	clientOpts := []client.Option{
		client.WithReadBufferCap(options.TransferClientReadBufferCap),
		client.WithWriteBufferCap(options.TransferClientWriteBufferCap),
		client.WithWriteQueueCap(options.TransferClientWriteQueueCap),
		client.WithTCPKeepAlive(options.TransferClientTCPKeepAlive),
		client.WithSocketRecvBuffer(options.TransferClientSocketRecvBuffer),
		client.WithSocketSendBuffer(options.TransferClientSocketSendBuffer),
	}
	tClient, err := client.NewClient(remoteSrv.codec, newRemoteListener(remoteSrv), clientOpts...)
	if err != nil {
		return err
	}
	if bErr, connCtx := remoteSrv.connectCtx.beforeConnect(); bErr != nil {
		return bErr
	} else {
		go remoteSrv._connect(tClient, connCtx)
	}
	return nil
}

// _connect
//
//	@Description: 真实连接操作
//	@receiver sClient
//	@param tClient
//	@param connCtx
func (remoteSrv *Remote) _connect(tClient *client.Client, connCtx context.Context) {
	address := fmt.Sprintf("%s:%d", remoteSrv.info.Ip, remoteSrv.info.Port)
	plog.Debug("transfer client try connecting", pfield.String("address", address))

	err := tClient.Dial(connCtx, address)
	if err != nil {
		plog.Error("transfer client Connect service error:",
			pfield.String("targetServiceId", remoteSrv.info.ServiceId()),
			pfield.String("targetAddress", address),
			pfield.Error(err))
	} else {
		plog.Debug("transfer client connected", pfield.String("address", address))
	}
	remoteSrv.connectCtx.afterConnect(tClient, err)
}

// handshake
//
//	@Description: 握手
//	@receiver sClient
func (remoteSrv *Remote) handshake() {
	if remoteSrv.certified.Load() {
		return
	}
	appInfo := remoteSrv.manager.transfer.AppInfo
	options := remoteSrv.manager.transfer.Options
	// 发送握手
	remoteSrv.inner.SendMessage(&codec.HandshakeReq{
		Id:        appInfo.Id(), // 当前服务Id
		AuthKey:   options.TransferServiceAuthKey,
		Service:   remoteSrv.info.Service(),   // 对方服务名
		ServiceId: remoteSrv.info.ServiceId(), // 对方实例Id
	})
}

func (remoteSrv *Remote) onHandshake() {
	remoteSrv.certified.CompareAndSwap(false, true)
	plog.Debug("(transfer client) on handshake", pfield.Any("info", remoteSrv.info))
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
		if remoteSrv.inner != nil {
			if remoteSrv.inner.IsClosed() {
				err := remoteSrv.Connect()
				if err != nil {
					plog.Error("reconnect error:", pfield.Error(err))
				}
			} else {
				if !remoteSrv.certified.Load() {
					remoteSrv.handshake()
				} else {
					// 连接正常则发送心跳
					remoteSrv.inner.SendMessage(&codec.HeartbeatSReq{})
				}
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
		return service.ErrDisabledService
	case StateStopped:
		return service.ErrStoppedInstance
	default:
	}
	if remoteSrv.inner.IsClosed() {
		err := remoteSrv.Connect()
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

func (remoteSrv *Remote) TransferMessage(msg []byte) error {
	if err := remoteSrv.checkAlive(); err != nil {
		return err
	}
	remoteSrv.inner.AsyncWrite(msg, func(c session.Conn, err error) error {
		if err != nil {
			plog.Error("write message error:", pfield.Error(err))
			return nil
		}
		return nil
	})
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
		return service.ErrStoppedInstance
	}
	// 尝试取消正在进行的连接
	remoteSrv.connectCtx.cancelConnect()
	if remoteSrv.inner != nil {
		if err := remoteSrv.inner.CloseWithContext(ctx); err != nil {
			plog.Error("close connection error:", pfield.Error(err))
		}
	}
	return nil
}
