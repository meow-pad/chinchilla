package transfer

import (
	"chinchilla/gateway"
	"chinchilla/transfer/codec"
	"chinchilla/transfer/selector"
	"chinchilla/transfer/service"
	"context"
	"encoding/binary"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"github.com/meow-pad/persian/frame/pservice/cache"
	"github.com/meow-pad/persian/utils/timewheel"
	"github.com/meow-pad/persian/utils/worker"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
)

type Transfer struct {
	Gateway *gateway.Gateway     `autowire:""`
	Timer   *timewheel.TimeWheel `autowire:"SecondTimer"`
	Cache   *cache.Cache         `autowire:""`

	selector       selector.Selector
	executor       *worker.FixedWorkerPool
	clientManagers map[string]*service.Manager
	cleanTask      *timewheel.Task
	keepAliveTask  *timewheel.Task
}

func (transfer *Transfer) init() (err error) {
	options := transfer.Gateway.Options
	if options.ServiceSelector == nil {
		transfer.selector = selector.NewCompositeSelector(
			selector.NewCacheSelector(transfer.Cache), selector.NewWeightSelector())
	} else {
		transfer.selector = options.ServiceSelector
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
	clientCodec := codec.NewClientCodec(binary.LittleEndian)
	transfer.clientManagers = make(map[string]*service.Manager)
	options := transfer.Gateway.Options
	for _, srvName := range options.RegistryServiceNames {
		if srvManager, sErr := service.NewManager(transfer, srvName, clientCodec, transfer.selector); sErr != nil {
			return sErr
		} else {
			transfer.clientManagers[srvName] = srvManager
		}
	}
	transfer.cleanTask = transfer.Timer.AddCron(options.CleanSenderSessionCacheInterval, transfer.cleanExpiredSessions)
	transfer.keepAliveTask = transfer.Timer.AddCron(options.TransferKeepAliveInterval, transfer.keepClientsAlive)
	return nil
}

func (transfer *Transfer) Stop(ctx context.Context) error {
	if err := transfer.Timer.Remove(transfer.keepAliveTask); err != nil {
		plog.Error("remove keepAliveTask error", pfield.Error(err))
	}
	if err := transfer.Timer.Remove(transfer.cleanTask); err != nil {
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
		manager := transfer.clientManagers[srvName]
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
		plog.Error("submit clean-expired-session-task error:", pfield.Error(err))
	}
}

// keepClientsAlive
//
//	@Description: 客户端保活
//	@receiver transfer
func (transfer *Transfer) keepClientsAlive() {
	for _, manager := range transfer.clientManagers {
		manager.KeepClientsAlive()
	}
}

func (transfer *Transfer) GetServiceManager(service string) *service.Manager {
	manager, _ := transfer.clientManagers[service]
	return manager
}
