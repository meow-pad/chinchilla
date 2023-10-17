package gateway

import (
	"github.com/meow-pad/chinchilla/handler"
	"github.com/meow-pad/chinchilla/transfer/selector"
	"github.com/meow-pad/persian/utils/runtime"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"time"
)

var options *Options

func NewOptions() *Options {
	return &Options{
		UnregisteredSenderExpiration:    15_000,
		RegisteredSenderExpiration:      30_000,
		CleanSenderSessionCacheInterval: 30 * time.Second,

		MessageExecutorWorkerNum:   runtime.NumGoroutine() + 1,
		MessageExecutorQueueLength: 1000,

		TransferClientReadBufferCap:    512 * 1024,
		TransferClientWriteBufferCap:   512 * 1024,
		TransferClientWriteQueueCap:    100,
		TransferClientTCPKeepAlive:     60 * time.Second,
		TransferClientSocketRecvBuffer: 512 * 1024,
		TransferClientSocketSendBuffer: 512 * 1024,
		TransferClientDialTimeout:      5 * time.Second,
		TransferClientDisableTimeout:   60_000,
		TransferKeepAliveInterval:      10 * time.Second,

		NamingServicePort:      8848,
		NamingServiceTimeoutMs: 10 * 1000,
		NamingServiceLogLevel:  "warn",
	}
}

type Options struct {
	// 认证
	ReceiverAuthKey string // setting

	// 过期时间，单位毫秒
	UnregisteredSenderExpiration int64
	// 过期时间，单位毫秒
	RegisteredSenderExpiration int64
	// 清理session缓存间隔
	CleanSenderSessionCacheInterval time.Duration

	// 工作核心数，默认逻辑核心数加1
	MessageExecutorWorkerNum int
	// 工作队列长度
	MessageExecutorQueueLength int

	// 转发读缓冲容量
	TransferClientReadBufferCap int
	// 转发写缓冲区
	TransferClientWriteBufferCap int
	// 转发写队列容量
	TransferClientWriteQueueCap int
	// 转发TCPKeepAlive
	TransferClientTCPKeepAlive time.Duration
	// 转发socket读缓冲
	TransferClientSocketRecvBuffer int
	// 转发socket写缓冲
	TransferClientSocketSendBuffer int
	// 连接超时
	TransferClientDialTimeout time.Duration
	// 关闭Disable客户端超时时间
	TransferClientDisableTimeout int64
	// 转发重连检查间隔
	TransferKeepAliveInterval time.Duration
	// 转发服务间认证信息
	TransferServiceAuthKey string // setting

	// 服务地址
	NamingServiceIPAddr string // setting
	// 服务端口，默认 8848
	NamingServicePort uint64
	// 命名空间
	NamingServiceNamespaceId string // setting
	// 连接超时
	NamingServiceTimeoutMs uint64
	// 日志目录
	NamingServiceLogDir string // setting
	// 日志等级，如“debug”、“info”
	NamingServiceLogLevel string
	// 日志归档
	NamingServiceLogRolling constant.ClientLogRollingConfig // setting
	// 缓存目录
	NamingServiceCacheDir string // setting
	// 用户名密码
	NamingServiceUsername string // setting
	NamingServicePassword string // setting

	// 关注的服务名
	RegistryServiceNames []string // setting
	// 服务消息处理器
	ServiceMessageHandler map[string]handler.MessageHandler // setting
	// 服务选择器
	ServiceSelector selector.Selector // 默认值为cache和wrr的组合
}
