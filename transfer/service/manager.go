package service

import (
	"errors"
	"github.com/meow-pad/chinchilla/transfer"
	"github.com/meow-pad/chinchilla/transfer/codec"
	"github.com/meow-pad/chinchilla/transfer/selector"
	"github.com/meow-pad/persian/errdef"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"reflect"
	"sync"
)

func NewManager(transfer *transfer.Transfer,
	service string, clientCodec *codec.ClientCodec, selector selector.Selector) (*Manager, error) {
	if transfer == nil || clientCodec == nil {
		return nil, errdef.ErrInvalidParams
	}
	manager := &Manager{transfer: transfer,
		service: service, clientCodec: clientCodec, selector: selector}
	return manager, nil
}

type Manager struct {
	transfer    *transfer.Transfer
	service     string
	selector    selector.Selector
	clientCodec *codec.ClientCodec

	services   sync.Map
	srvInfoArr []Info
}

// UpdateInstances
//
//	@Description: 更新服务实例信息
//	@receiver manager
//	@param instArr
func (manager *Manager) UpdateInstances(instArr []model.Instance) {
	var infoMap map[string]*Info
	var srvInfoArr []Info
	for _, inst := range instArr {
		info := Info(inst)
		infoMap[info.InstanceId] = &info
		// 添加新增的服务的连接
		_, ok := manager.services.Load(info.InstanceId)
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
		} // end of if
	}
	manager.services.Range(func(key, value any) bool {
		id := key.(string)
		srv := value.(Service)
		newInst, _ := infoMap[id]
		if newInst == nil {
			cInfo := srv.Info()
			cInfo.Enable = false
			newInst = &cInfo
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
	manager.selector.Update(manager.srvInfoArr)
}

// addService
//
//	@Description: 新增服务
//	@receiver manager
//	@param inst
//	@return error
func (manager *Manager) addService(info Info) error {
	curInstId := manager.transfer.AppInfo.Id()
	if curInstId == info.InstanceId {
		if srv, err := newLocalService(manager, info); err != nil {
			return err
		} else {
			manager.services.Store(info.InstanceId, srv)
		}
	} else {
		srv, err := newRemoteService(manager, info)
		if err != nil {
			return err
		}
		manager.services.Store(info.InstanceId, srv)
		// 尝试连接
		err = srv.connect()
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
	manager.services.Range(func(key, val any) bool {
		if !val.(Service).KeepAlive() {
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
func (manager *Manager) SelectInstance(routerId string) (Service, error) {
	instId, err := manager.selector.Select(routerId)
	if err != nil && !errors.Is(err, selector.ErrEmptyInstances) {
		return nil, err
	}
	if len(instId) <= 0 {
		return nil, nil
	}
	srvObj, _ := manager.services.Load(instId)
	if srvObj == nil {
		plog.Warn("cannot find client by instance id")
		return nil, nil
	}
	client := srvObj.(Service)
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
//	@return []model.Info
func (manager *Manager) GetServiceInfoArray() []Info {
	return manager.srvInfoArr
}
