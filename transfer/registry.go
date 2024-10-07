package transfer

import (
	"context"
	"errors"
	"github.com/meow-pad/chinchilla/option"
	"github.com/meow-pad/persian/frame/pboot"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/frame/pservice/name"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"reflect"
	"sync"
)

func NewRegistry(appInfo pboot.AppInfo, transfer *Transfer, options *option.Options) (*Registry, error) {
	registry := &Registry{
		appInfo:  appInfo,
		transfer: transfer,
		options:  options,
	}
	if err := registry.init(); err != nil {
		return nil, err
	}
	return registry, nil
}

type Registry struct {
	appInfo  pboot.AppInfo
	options  *option.Options
	transfer *Transfer

	naming     *name.NacosNaming
	autonomous bool
	services   sync.Map
}

func (registry *Registry) init() error {
	options := registry.options
	if options.NamingService != nil {
		registry.naming = options.NamingService
	} else {
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
		if registry.naming, err = name.NewNacosNaming(cConfig, sConfig); err != nil {
			return err
		}
		registry.autonomous = true
	}
	return nil
}

func (registry *Registry) Start(ctx context.Context) error {
	options := registry.options
	for _, srvName := range options.RegistryServiceNames {
		if err := registry.initService(srvName); err != nil {
			return err
		}
		if err := registry.subscribeService(srvName); err != nil {
			return err
		}
	}
	return nil
}

func (registry *Registry) Stop(ctx context.Context) error {
	options := registry.options
	for _, srvName := range options.RegistryServiceNames {
		if err := registry.UnSubscribeService(srvName); err != nil {
			plog.Error("unsubscribe service error:", pfield.Error(err))
		}
	}
	if registry.autonomous {
		registry.naming.CloseClient()
	}
	return nil
}

// initService
//
//	@Description: 初始化服务当前所有实例
//	@receiver registry
//	@param srv
//	@return error
func (registry *Registry) initService(srv string) error {
	if _, ok := registry.services.Load(srv); ok {
		return nil
	}
	params := vo.GetServiceParam{
		//Clusters:    []string{registry.appInfo.Cluster()},
		ServiceName: srv,
		GroupName:   registry.appInfo.NamingGroup(),
	}
	srvModel, err := registry.naming.GetService(params)
	if err != nil {
		return err
	}
	cluster := registry.appInfo.Cluster()
	cInstances := make([]model.Instance, 0, len(srvModel.Hosts))
	for _, inst := range srvModel.Hosts {
		if inst.ClusterName == cluster {
			cInstances = append(cInstances, inst)
		}
	}
	plog.Debug("init services:", pfield.JsonString("instances", cInstances))
	registry.transfer.UpdateInstances(srv, cInstances)
	return nil
}

// subscribeService
//
//	@Description: 订阅服务
//	@receiver manager
//	@param srv
//	@return error
func (registry *Registry) subscribeService(srv string) error {
	params := &vo.SubscribeParam{
		//Clusters:    []string{registry.appInfo.Cluster()},
		ServiceName: srv,
		GroupName:   registry.appInfo.NamingGroup(),
		SubscribeCallback: func(instances []model.Instance, err error) {
			cluster := registry.appInfo.Cluster()
			cInstances := make([]model.Instance, 0, len(instances))
			for _, inst := range instances {
				if inst.ClusterName == cluster {
					cInstances = append(cInstances, inst)
				}
			}
			plog.Debug("on services changed:", pfield.JsonString("instances", cInstances))
			registry.transfer.UpdateInstances(srv, cInstances)
		},
	}
	err := registry.naming.Subscribe(params)
	if err != nil {
		return err
	}
	registry.services.Store(srv, params)
	return nil
}

// UnSubscribeService
//
//	@Description: 取消订阅
//	@receiver manager
//	@param srv
//	@return error
func (registry *Registry) UnSubscribeService(srv string) error {
	var (
		value  any
		ok     bool
		params *vo.SubscribeParam
	)
	if value, ok = registry.services.Load(srv); ok {
		return nil
	}
	if params, ok = value.(*vo.SubscribeParam); !ok || params == nil {
		return errors.New("invalid params:" + reflect.TypeOf(value).String())
	}
	err := registry.naming.Unsubscribe(params)
	if err != nil {
		return err
	}
	registry.services.Delete(srv)
	return nil
}

// getService
//
//	@Description: get service
//	@receiver manager
//	@param srv
//	@return model.Service
//	@return error
func (registry *Registry) getService(srv string) (model.Service, error) {
	return registry.naming.GetService(vo.GetServiceParam{
		Clusters:    []string{registry.appInfo.Cluster()},
		ServiceName: srv,
		GroupName:   registry.appInfo.NamingGroup(),
	})
}

// selectInstances
//
//	@Description: only return the instances of healthy=true,enable=true and weight>0
//	@receiver manager
//	@param srv
//	@return []model.Info
//	@return error
func (registry *Registry) selectInstances(srv string) ([]model.Instance, error) {
	return registry.naming.SelectInstances(vo.SelectInstancesParam{
		Clusters:    []string{registry.appInfo.Cluster()},
		ServiceName: srv,
		GroupName:   registry.appInfo.NamingGroup(),
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
func (registry *Registry) selectOneHealthInstance(srv string) (*model.Instance, error) {
	return registry.naming.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		Clusters:    []string{registry.appInfo.Cluster()},
		ServiceName: srv,
		GroupName:   registry.appInfo.NamingGroup(),
	})
}
