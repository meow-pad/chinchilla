package trtest

import (
	"context"
	"errors"
	"github.com/meow-pad/chinchilla/transfer/common"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

func newTSNaming(appInfo *AppInfo, naming *NacosNaming) *TSNaming {
	return &TSNaming{
		TSAppInfo: appInfo,
		Naming:    naming,
	}
}

type TSNaming struct {
	TSAppInfo *AppInfo
	Naming    *NacosNaming
}

func (naming *TSNaming) Start() error {
	// 注册服务
	if result, err := naming.Naming.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          naming.TSAppInfo.IP(),
		Port:        naming.TSAppInfo.Port(),
		Weight:      1,
		Enable:      true,
		Healthy:     true,
		Metadata:    naming.buildMetadata(),
		ClusterName: naming.TSAppInfo.Cluster(),
		ServiceName: naming.TSAppInfo.Name(),
		GroupName:   naming.TSAppInfo.NamingGroup(),
		Ephemeral:   true,
	}); err != nil {
		return err
	} else if !result {
		return errors.New("fail to Register")
	}
	return nil
}

func (naming *TSNaming) Stop(cxt context.Context) error {
	if success, err := naming.Naming.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          naming.TSAppInfo.IP(),
		Port:        naming.TSAppInfo.Port(),
		Cluster:     naming.TSAppInfo.Cluster(),
		ServiceName: naming.TSAppInfo.Name(),
		GroupName:   naming.TSAppInfo.NamingGroup(),
		Ephemeral:   true,
	}); err != nil {
		return err
	} else if !success {
		plog.Error("fail to Deregister TSNaming")
	}
	return nil
}

func (naming *TSNaming) buildMetadata() map[string]string {
	return map[string]string{
		common.MetadataKeyId: naming.TSAppInfo.Id(),
	}
}
