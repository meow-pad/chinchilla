package trtest

import (
	"github.com/meow-pad/persian/frame/pservice/name"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

type NacosOptions struct {
	Ip          string
	Port        uint64
	NamespaceId string
	Username    string
	Password    string
}

func newNacosNaming(options *NacosOptions) (*NacosNaming, error) {
	naming := &NacosNaming{
		options: options,
	}
	if err := naming.init(); err != nil {
		return nil, err
	}
	return naming, nil
}

type NacosNaming struct {
	*name.NacosNaming

	options *NacosOptions
}

func (naming *NacosNaming) init() error {
	sConfig := []constant.ServerConfig{
		*constant.NewServerConfig(naming.options.Ip, naming.options.Port),
	}
	cConfig := constant.NewClientConfig(
		constant.WithNamespaceId(naming.options.NamespaceId),
		constant.WithTimeoutMs(10_000),
		constant.WithNotLoadCacheAtStart(true),
		//constant.WithLogDir(path.Join(nacosCfg.LogDir, "naming")),
		//constant.WithLogLevel(nacosCfg.LogLevel),
		//constant.WithLogRollingConfig(&constant.ClientLogRollingConfig{
		//	MaxAge:    nacosCfg.LogMaxAge,
		//	LocalTime: true,
		//}),
		//constant.WithCacheDir(path.Join(nacosCfg.CacheDir, "config")),
		constant.WithUsername(naming.options.Username),
		constant.WithPassword(naming.options.Password),
	)
	var err error
	naming.NacosNaming, err = name.NewNacosNaming(cConfig, sConfig)
	if err != nil {
		return err
	}
	return nil
}

func (naming *NacosNaming) Start() error {
	return nil
}

func (naming *NacosNaming) Stop() error {
	if naming.NacosNaming != nil {
		naming.NacosNaming.CloseClient()
	}
	return nil
}
