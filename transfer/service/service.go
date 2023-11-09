package service

import (
	"context"
	"errors"
	"github.com/meow-pad/chinchilla/transfer/common"
)

var (
	ErrDisabledService = errors.New("service is disabled")
	ErrStoppedInstance = errors.New("instance is stopped")
)

type Service interface {
	// UpdateInfo
	//  @Description: 更新服务实例状态
	//  @param info
	//  @return error
	//
	UpdateInfo(info common.Info) error

	// Info
	//  @Description: 状态对象
	//  @return model.Info
	//
	Info() common.Info

	// KeepAlive
	//  @Description: 保活操作
	//  @return bool
	//
	KeepAlive() bool

	// SendMessage
	//  @Description: 发送消息
	//  @param msg
	//  @return error
	//
	SendMessage(msg any) error

	// IsEnable
	//  @Description: 服务是否处于可用状态
	//  @return bool
	//
	IsEnable() bool

	// Stop
	//  @Description: 停止服务
	//  @param ctx
	//  @return error
	//
	Stop(ctx context.Context) error

	// IsStopped
	//  @Description: 是否处于停止状态
	//  @return bool
	//
	IsStopped() bool
}
