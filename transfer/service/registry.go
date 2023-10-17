package service

import (
	"chinchilla/gateway"
	"chinchilla/transfer"
	"context"
	"errors"
	"github.com/meow-pad/persian/frame/pboot"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pservice/name"
	"github.com/meow-pad/persian/utils/json"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"reflect"
	"sync"
)

type Registry struct {
	AppInfo  pboot.AppInfo      `autowire:""`
	Gateway  *gateway.Gateway   `autowire:""`
	Transfer *transfer.Transfer `autowire:""`

	naming   *name.NacosNaming
	services sync.Map
}

func (manager *Registry) init() error {
	options := manager.Gateway.Options
	sConfig := []constant.ServerConfig{
		*constant.NewServerConfig(options.NamingServiceIPAddr, options.NamingServicePort),
	}
	cConfig := constant.NewClientConfig(
		constant.WithNamespaceId(options.NamingServiceNamespaceId),
		constant.WithTimeoutMs(options.NamingServiceTimeoutMs),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir(options.NamingServiceLogDir),
		constant.WithLogLevel(options.NamingServiceLogLevel),
		constant.WithLogRollingConfig(&options.NamingServiceLogRolling),
		constant.WithCacheDir(options.NamingServiceCacheDir),
		constant.WithUsername(options.NamingServiceUsername),
		constant.WithPassword(options.NamingServicePassword),
	)
	var err error
	if manager.naming, err = name.NewNacosNaming(cConfig, sConfig); err != nil {
		return err
	}
	return nil
}

func (manager *Registry) Start(ctx context.Context) error {
	options := manager.Gateway.Options
	for _, srvName := range options.RegistryServiceNames {
		if err := manager.SubscribeService(srvName); err != nil {
			return err
		}
	}
	return nil
}

func (manager *Registry) Stop(ctx context.Context) error {
	options := manager.Gateway.Options
	for _, srvName := range options.RegistryServiceNames {
		if err := manager.UnSubscribeService(srvName); err != nil {
			plog.Error("unsubscribe service error:", pfield.Error(err))
		}
	}
	return nil
}

// SubscribeService
//
//	@Description: 订阅服务
//	@receiver manager
//	@param srv
//	@return error
func (manager *Registry) SubscribeService(srv string) error {
	if _, ok := manager.services.Load(srv); ok {
		return nil
	}
	params := &vo.SubscribeParam{
		Clusters:    []string{manager.AppInfo.Cluster()},
		ServiceName: srv,
		GroupName:   manager.AppInfo.EnvName(),
		SubscribeCallback: func(instances []model.Instance, err error) {
			plog.Debug("on services changed:", pfield.String("instances", json.ToString(instances)))
			manager.Transfer.UpdateInstances(srv, instances)
		},
	}
	err := manager.naming.Subscribe(params)
	if err != nil {
		return err
	}
	manager.services.Store(srv, params)
	return nil
}

// UnSubscribeService
//
//	@Description: 取消订阅
//	@receiver manager
//	@param srv
//	@return error
func (manager *Registry) UnSubscribeService(srv string) error {
	var (
		value  any
		ok     bool
		params *vo.SubscribeParam
	)
	if value, ok = manager.services.Load(srv); ok {
		return nil
	}
	if params, ok = value.(*vo.SubscribeParam); !ok || params == nil {
		return errors.New("invalid params:" + reflect.TypeOf(value).String())
	}
	err := manager.naming.Unsubscribe(params)
	if err != nil {
		return err
	}
	manager.services.Delete(srv)
	return nil
}

// getService
//
//	@Description: get service
//	@receiver manager
//	@param srv
//	@return model.Service
//	@return error
func (manager *Registry) getService(srv string) (model.Service, error) {
	return manager.naming.GetService(vo.GetServiceParam{
		Clusters:    []string{manager.AppInfo.Cluster()},
		ServiceName: srv,
		GroupName:   manager.AppInfo.EnvName(),
	})
}

// selectInstances
//
//	@Description: only return the instances of healthy=true,enable=true and weight>0
//	@receiver manager
//	@param srv
//	@return []model.Info
//	@return error
func (manager *Registry) selectInstances(srv string) ([]model.Instance, error) {
	return manager.naming.SelectInstances(vo.SelectInstancesParam{
		Clusters:    []string{manager.AppInfo.Cluster()},
		ServiceName: srv,
		GroupName:   manager.AppInfo.EnvName(),
		HealthyOnly: true,
	})
}

// selectOneHealthInstance
//
//	@Description: return one instance by WRR strategy for load balance
//		And the instance should be health=true,enable=true and weight>0
//	@receiver manager
//	@param srv
//	@return *model.Info
//	@return error
func (manager *Registry) selectOneHealthInstance(srv string) (*model.Instance, error) {
	return manager.naming.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		Clusters:    []string{manager.AppInfo.Cluster()},
		ServiceName: srv,
		GroupName:   manager.AppInfo.EnvName(),
	})
}
