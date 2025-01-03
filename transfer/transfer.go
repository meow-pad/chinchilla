package transfer

import (
	"context"
	"github.com/meow-pad/chinchilla/option"
	"github.com/meow-pad/chinchilla/transfer/codec"
	"github.com/meow-pad/chinchilla/transfer/router"
	"github.com/meow-pad/chinchilla/transfer/selector"
	"github.com/meow-pad/chinchilla/utils/gopool"
	"github.com/meow-pad/persian/frame/pboot"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"github.com/meow-pad/persian/utils/timewheel"
	"github.com/meow-pad/persian/utils/worker"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
)

func NewTransfer(
	appInfo pboot.AppInfo,
	secTimer *timewheel.TimeWheel,
	goPool *gopool.GoPool,
	options *option.Options,
) (*Transfer, error) {
	transfer := &Transfer{
		AppInfo:  appInfo,
		Options:  options,
		SecTimer: secTimer,
		GoPool:   goPool,
	}
	err := transfer.init()
	if err != nil {
		return nil, err
	}
	return transfer, nil
}

type Transfer struct {
	AppInfo  pboot.AppInfo
	Options  *option.Options
	SecTimer *timewheel.TimeWheel
	GoPool   *gopool.GoPool

	registry      *Registry
	selector      selector.Selector
	router        router.Router
	executor      *worker.FixedWorkerPool
	clientMgrMap  map[string]*Manager
	cleanTask     *timewheel.Task
	keepAliveTask *timewheel.Task
}

func (transfer *Transfer) init() (err error) {
	options := transfer.Options
	transfer.registry, err = NewRegistry(transfer.AppInfo, transfer, transfer.Options)
	if err != nil {
		return err
	}
	if options.ServiceSelector == nil {
		transfer.selector = selector.NewWeightSelector()
	} else {
		transfer.selector = options.ServiceSelector
	}
	if options.ServiceRouter == nil {
		transfer.router = &router.CommonRouter{}
	} else {
		transfer.router = options.ServiceRouter
	}
	if transfer.executor, err = worker.NewFixedWorkerPool(
		options.MessageExecutorWorkerNum,
		options.MessageExecutorQueueLength,
		true,
	); err != nil {
		return
	}
	return
}

func (transfer *Transfer) Start(ctx context.Context) error {
	clientCodec := codec.NewClientCodec(codec.MessageCodecByteOrder)
	transfer.clientMgrMap = make(map[string]*Manager)
	options := transfer.Options
	for _, srvName := range options.RegistryServiceNames {
		if srvManager, sErr := NewManager(transfer, srvName, clientCodec); sErr != nil {
			return sErr
		} else {
			transfer.clientMgrMap[srvName] = srvManager
		}
	}
	transfer.cleanTask = transfer.SecTimer.AddCron(options.CleanSenderSessionCacheInterval, transfer.cleanExpiredSessions)
	transfer.keepAliveTask = transfer.SecTimer.AddCron(options.TransferKeepAliveInterval, transfer.keepClientsAlive)
	if err := transfer.registry.Start(ctx); err != nil {
		return err
	}
	return nil
}

func (transfer *Transfer) Stop(ctx context.Context) error {
	if err := transfer.registry.Stop(ctx); err != nil {
		plog.Error("stop registry error:", pfield.Error(err))
	}
	if err := transfer.SecTimer.Remove(transfer.keepAliveTask); err != nil {
		plog.Error("remove keepAliveTask error", pfield.Error(err))
	}
	if err := transfer.SecTimer.Remove(transfer.cleanTask); err != nil {
		plog.Error("remove cleanTask error", pfield.Error(err))
	}
	return nil
}

// Forward
//
//	@Description: 指定连接的任务处理
//	@receiver transfer
//	@param connId
//	@param task
func (transfer *Transfer) Forward(connId int64, task func(*worker.GoroutineLocal)) {
	if err := transfer.executor.Submit(int(connId), task); err != nil {
		plog.Error("forward task error:", pfield.Error(err))
	}
}

// UpdateInstances
//
//	@Description: 更新服务实例
//	@receiver transfer
//	@param srvName
//	@param instances
func (transfer *Transfer) UpdateInstances(srvName string, instances []model.Instance) {
	if err := transfer.executor.Submit(0, func(*worker.GoroutineLocal) {
		manager := transfer.clientMgrMap[srvName]
		if manager == nil {
			plog.Error("unknown service", pfield.String("srvName", srvName))
		} else {
			manager.UpdateInstances(instances)
		}
	}); err != nil {
		plog.Error("submit change-service-task error:", pfield.Error(err))
	}
}

// cleanExpiredSessions
//
//	@Description: 清理本地缓存中过期session
//	@receiver transfer
func (transfer *Transfer) cleanExpiredSessions() {
	if err := transfer.executor.SubmitToAll(func(local *worker.GoroutineLocal) {
		var toDelete []any
		local.Range(func(key, val any) bool {
			sess := val.(session.Session)
			if sess != nil {
				// 已经关闭的连接需要清理
				if sess.IsClosed() {
					toDelete = append(toDelete, key)
				}
			}
			return true
		})
		for _, key := range toDelete {
			local.Remove(key)
		}
	}, false); err != nil {
		plog.Error("submit clean-expired-serverSession-task error:", pfield.Error(err))
	}
}

// keepClientsAlive
//
//	@Description: 客户端保活
//	@receiver transfer
func (transfer *Transfer) keepClientsAlive() {
	for _, manager := range transfer.clientMgrMap {
		manager.KeepClientsAlive()
	}
}

func (transfer *Transfer) GetServiceManager(service string) *Manager {
	manager, _ := transfer.clientMgrMap[service]
	return manager
}
