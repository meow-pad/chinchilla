package transfer

import (
	"errors"
	"fmt"
	"github.com/meow-pad/chinchilla/transfer/codec"
	"github.com/meow-pad/chinchilla/transfer/common"
	"github.com/meow-pad/chinchilla/transfer/service"
	"github.com/meow-pad/persian/errdef"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/utils/coding"
	"github.com/meow-pad/persian/utils/collections"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"reflect"
)

func NewManager(transfer *Transfer,
	service string, clientCodec *codec.ClientCodec) (*Manager, error) {
	if transfer == nil || clientCodec == nil {
		return nil, errdef.ErrInvalidParams
	}
	manager := &Manager{transfer: transfer,
		service: service, clientCodec: clientCodec}
	return manager, nil
}

type Manager struct {
	transfer    *Transfer
	service     string
	clientCodec *codec.ClientCodec

	services   collections.SyncMap[string, service.Service]
	srvInfoArr []common.Info
}

// UpdateInstances
//
//	@Description: 更新服务实例信息
//	@receiver manager
//	@param instArr
func (manager *Manager) UpdateInstances(instArr []model.Instance) {
	var infoMap = make(map[string]*common.Info)
	var srvInfoArr []common.Info
	for _, inst := range instArr {
		info := common.Info(inst)
		serviceId := info.ServiceId()
		// 添加新增的服务的连接
		_, ok := manager.services.Load(serviceId)
		if !ok {
			if !info.Enable || !info.Healthy {
				// 无法使用的服务无需添加
				continue
			}
			if err := manager.addService(info); err != nil {
				plog.Error("cant build service", pfield.Error(err))
			} else {
				// 仅记录可用的实例
				srvInfoArr = append(srvInfoArr, info)
			}
			infoMap[serviceId] = nil // 新加进来的仅标识
		} else {
			infoMap[serviceId] = &info
		}
	}
	manager.services.Range(func(id string, srv service.Service) bool {
		newInst, exist := infoMap[id]
		if newInst == nil {
			if exist {
				// 新加进来的直接返回
				return true
			} else {
				cInfo := srv.Info()
				cInfo.Enable = false
				newInst = &cInfo
			}
		}
		if err := srv.UpdateInfo(*newInst); err != nil {
			plog.Error("update service instance error:", pfield.Error(err))
		}
		if srv.IsEnable() {
			// 仅记录可用的实例
			srvInfoArr = append(srvInfoArr, srv.Info())
		}
		return true
	})
	manager.srvInfoArr = srvInfoArr
	manager.transfer.selector.Update(manager.srvInfoArr)
}

// addService
//
//	@Description: 新增服务
//	@receiver manager
//	@param inst
//	@return error
func (manager *Manager) addService(info common.Info) error {
	curInstId := manager.transfer.AppInfo.Id()
	srvId := info.ServiceId()
	if len(srvId) <= 0 {
		return fmt.Errorf("less metadata service id:%v", info)
	}
	if curInstId == srvId {
		if srv, err := NewLocalService(manager, info); err != nil {
			return err
		} else {
			manager.services.Store(srvId, srv)
		}
	} else {
		srv, err := NewRemoteService(manager, info)
		if err != nil {
			return err
		}
		manager.services.Store(srvId, srv)
		// 尝试连接
		err = srv.Connect()
		if err != nil {
			plog.Error("connect to service error:", pfield.Error(err))
		}
	}
	return nil
}

// KeepClientsAlive
//
//	@Description: 保活操作
//	@receiver manager
func (manager *Manager) KeepClientsAlive() {
	manager.services.Range(func(key string, val service.Service) bool {
		if !val.(service.Service).KeepAlive() {
			manager.services.Delete(key)
		}
		return true
	})
}

// SelectInstance
//
//	@Description: 选择可用服务
//	@receiver manager
//	@param routerId
//	@return Service
//	@return error
func (manager *Manager) SelectInstance(routerId uint64) (service.Service, error) {
	defer coding.CachePanicError("select service instance error", nil,
		pfield.Uint64("routerId", routerId))
	instId, err := manager.transfer.selector.Select(routerId)
	if err != nil && !errors.Is(err, common.ErrEmptyInstances) {
		return nil, err
	}
	return manager.getOpenService(instId)
}

// Route
//
//	@Description: 路由至可用服务
//	@receiver manager
//	@param routerType
//	@param routerId
//	@param msg
//	@return error
func (manager *Manager) Route(routerType int16, routerId string, msg []byte) error {
	return manager.transfer.router.Route(&manager.services, routerType, routerId, msg)
}

func (manager *Manager) getOpenService(instId string) (service.Service, error) {
	if len(instId) <= 0 {
		return nil, nil
	}
	srvObj, _ := manager.services.Load(instId)
	if srvObj == nil {
		plog.Warn("cannot find client by instance id")
		return nil, nil
	}
	client := srvObj.(service.Service)
	if client == nil {
		plog.Error("invalid client object", pfield.String("objType", reflect.TypeOf(srvObj).String()))
		return nil, nil
	}
	if client.IsStopped() {
		// 当前服务被关闭了（在注册中心丢失信息后被关闭），则不应该被返回；但这种情况在目前机制下不应该发生
		plog.Warn("a stopped service client is selected")
		return nil, nil
	}
	return client, nil
}

// GetServiceInfoArray
//
//	@Description: 获取实例信息
//	@receiver manager
//	@return []common.Info
func (manager *Manager) GetServiceInfoArray() []common.Info {
	return manager.srvInfoArr
}
