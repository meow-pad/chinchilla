package trtest

import (
	"context"
	"fmt"
	"github.com/meow-pad/chinchilla/gateway"
	"github.com/meow-pad/chinchilla/option"
	"github.com/meow-pad/persian/frame/pnet/tcp/session"
	"github.com/meow-pad/persian/frame/pnet/ws/server"
	"github.com/panjf2000/gnet/v2"
	"time"
)

type GatewayOptions struct {
	GWAppInfo *AppInfo
	IP        string
	Port      uint64
}

func newGateway(runtime *TransferRuntime, gwOptions *GatewayOptions) (*Gateway, error) {
	var naming *NacosNaming
	if nacosNaming, err := newNacosNaming(runtime.NacosOptions); err != nil {
		return nil, err
	} else {
		naming = nacosNaming
	}
	options := option.NewOptions(
		option.WithReceiverHandshakeAuthKey(UserHandshakeAuth),
		option.WithReceiverServerProtoAddr(fmt.Sprintf("tcp://%s:%d", gwOptions.IP, gwOptions.Port)),
		option.WithReceiverServerOptions(server.WithGNetOption(
			gnet.WithReadBufferCap(8*1024),
			gnet.WithWriteBufferCap(16*1024),
			gnet.WithSocketRecvBuffer(8*1024),
			gnet.WithSocketSendBuffer(16*1024),
		)),
		option.WithMessageExecutorQueueLength(2000),
		option.WithTransferClientWriteQueueCap(500),
		option.WithTransferClientDialTimeout(8*time.Second),
		option.WithTransferServiceAuthKey(ServerHandshakeAuth),
		// naming
		option.WithNamingService(naming.NacosNaming),
		// GoroutinePool
		option.WithGoroutinePool(runtime.GOPool),
		// 关注的服务名
		option.WithRegistryServiceNames(runtime.ServiceNames),
		// 服务消息处理器
		option.WithLocalContextBuilder(func(sess session.Session) (session.Context, error) {
			return nil, fmt.Errorf("cant create local context")
		}),
		//option.WithLocalServerCodec(common.ServerCodec),
		//option.WithLocalMessageHandler(map[string]handler.MessageHandler{
		//	starter.AppInfo.Name(): starter.MsgHandler,
		//}),
		// 服务选择器
		//option.WithServiceSelector(starter.Selector),
		// 服务路由器
		//option.WithServiceRouter(starter.Router),
	)
	gw, err := gateway.NewGateway(gwOptions.GWAppInfo, runtime.Timer, options)
	if err != nil {
		return nil, err
	}
	return &Gateway{
		Runtime: runtime,
		Options: gwOptions,
		gw:      gw,
		naming:  naming,
	}, nil
}

type Gateway struct {
	Runtime *TransferRuntime
	Options *GatewayOptions

	gw     *gateway.Gateway
	naming *NacosNaming
}

func (gateway *Gateway) Start() error {
	if err := gateway.naming.Start(); err != nil {
		return err
	}
	if err := gateway.gw.Start(context.Background()); err != nil {
		return err
	}
	return nil
}

func (gateway *Gateway) Stop(ctx context.Context) error {
	if err := gateway.gw.Stop(ctx); err != nil {
		return err
	}
	if err := gateway.naming.Stop(); err != nil {
		return err
	}
	return nil
}
