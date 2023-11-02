package option

import (
	"github.com/meow-pad/chinchilla/handler"
	"github.com/meow-pad/chinchilla/transfer/selector"
	ws "github.com/meow-pad/persian/frame/pnet/ws/server"
	"github.com/meow-pad/persian/frame/pservice/name"
	"github.com/meow-pad/persian/utils/runtime"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"time"
)

func NewOptions(opts ...Option) *Options {
	options := &Options{
		UnregisteredSenderExpiration:    20_000,
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
		TransferMessageWarningSize:     8 * 1024,

		NamingServicePort:      8848,
		NamingServiceTimeoutMs: 10 * 1000,
		NamingServiceLogLevel:  "warn",
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

type Options struct {
	// 认证
	ReceiverHandshakeAuthKey string // setting
	// 监听地址
	ReceiverServerProtoAddr string // setting
	// 服务器选项
	ReceiverServerOptions []ws.Option // setting

	// 为登录过期时间，单位毫秒
	UnregisteredSenderExpiration int64
	// 已登录过期时间，单位毫秒
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
	// 转发告警消息大小
	TransferMessageWarningSize int

	// 通过该配置直接配置服务或者通过以下配置创建一个
	NamingService *name.NacosNaming
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

type Option func(*Options)

func WithReceiverHandshakeAuthKey(value string) Option {
	return func(options *Options) {
		options.ReceiverHandshakeAuthKey = value
	}
}

func WithReceiverServerProtoAddr(value string) Option {
	return func(options *Options) {
		options.ReceiverHandshakeAuthKey = value
	}
}

func WithReceiverServerOptions(value ...ws.Option) Option {
	return func(options *Options) {
		options.ReceiverServerOptions = value
	}
}

func WithUnregisteredSenderExpiration(value int64) Option {
	return func(options *Options) {
		options.UnregisteredSenderExpiration = value
	}
}
func WithRegisteredSenderExpiration(value int64) Option {
	return func(options *Options) {
		options.RegisteredSenderExpiration = value
	}
}
func WithCleanSenderSessionCacheInterval(value time.Duration) Option {
	return func(options *Options) {
		options.CleanSenderSessionCacheInterval = value
	}
}
func WithMessageExecutorWorkerNum(value int) Option {
	return func(options *Options) {
		options.MessageExecutorWorkerNum = value
	}
}
func WithMessageExecutorQueueLength(value int) Option {
	return func(options *Options) {
		options.MessageExecutorQueueLength = value
	}
}
func WithTransferClientReadBufferCap(value int) Option {
	return func(options *Options) {
		options.TransferClientReadBufferCap = value
	}
}
func WithTransferClientWriteBufferCap(value int) Option {
	return func(options *Options) {
		options.TransferClientWriteBufferCap = value
	}
}
func WithTransferClientWriteQueueCap(value int) Option {
	return func(options *Options) {
		options.TransferClientWriteQueueCap = value
	}
}
func WithTransferClientTCPKeepAlive(value time.Duration) Option {
	return func(options *Options) {
		options.TransferClientTCPKeepAlive = value
	}
}
func WithTransferClientSocketRecvBuffer(value int) Option {
	return func(options *Options) {
		options.TransferClientSocketRecvBuffer = value
	}
}
func WithTransferClientSocketSendBuffer(value int) Option {
	return func(options *Options) {
		options.TransferClientSocketSendBuffer = value
	}
}
func WithTransferClientDialTimeout(value time.Duration) Option {
	return func(options *Options) {
		options.TransferClientDialTimeout = value
	}
}
func WithTransferClientDisableTimeout(value int64) Option {
	return func(options *Options) {
		options.TransferClientDisableTimeout = value
	}
}
func WithTransferKeepAliveInterval(value time.Duration) Option {
	return func(options *Options) {
		options.TransferKeepAliveInterval = value
	}
}
func WithTransferServiceAuthKey(value string) Option {
	return func(options *Options) {
		options.TransferServiceAuthKey = value
	}
}

func WithNamingService(value *name.NacosNaming) Option {
	return func(options *Options) {
		options.NamingService = value
	}
}

func WithNamingServiceIPAddr(value string) Option {
	return func(options *Options) {
		options.NamingServiceIPAddr = value
	}
}
func WithNamingServicePort(value uint64) Option {
	return func(options *Options) {
		options.NamingServicePort = value
	}
}
func WithNamingServiceNamespaceId(value string) Option {
	return func(options *Options) {
		options.NamingServiceNamespaceId = value
	}
}
func WithNamingServiceTimeoutMs(value uint64) Option {
	return func(options *Options) {
		options.NamingServiceTimeoutMs = value
	}
}
func WithNamingServiceLogDir(value string) Option {
	return func(options *Options) {
		options.NamingServiceLogDir = value
	}
}
func WithNamingServiceLogLevel(value string) Option {
	return func(options *Options) {
		options.NamingServiceLogLevel = value
	}
}
func WithNamingServiceLogRolling(value constant.ClientLogRollingConfig) Option {
	return func(options *Options) {
		options.NamingServiceLogRolling = value
	}
}

func WithNamingServiceCacheDir(value string) Option {
	return func(options *Options) {
		options.NamingServiceCacheDir = value
	}
}

func WithNamingServiceUsername(value string) Option {
	return func(options *Options) {
		options.NamingServiceUsername = value
	}
}
func WithNamingServicePassword(value string) Option {
	return func(options *Options) {
		options.NamingServicePassword = value
	}
}
func WithRegistryServiceNames(value []string) Option {
	return func(options *Options) {
		options.RegistryServiceNames = value
	}
}

func WithServiceMessageHandler(value map[string]handler.MessageHandler) Option {
	return func(options *Options) {
		options.ServiceMessageHandler = value
	}
}

func WithServiceSelector(value selector.Selector) Option {
	return func(options *Options) {
		options.ServiceSelector = value
	}
}
