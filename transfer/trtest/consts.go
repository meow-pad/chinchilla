package trtest

import "time"

const (
	UIDInvalid = 0

	RPCTimeout = 5 * time.Second

	UserHandshakeAuth   = "123"
	ServerHandshakeAuth = "123456"

	SessionContextIdInvalid = 0
)

const (
	RpcCodeSuccess = iota
)

const (
	ServerCodeInvalidTransferId = iota + 1
	ServerCodeInvalidAuth
	ServerCodeInvalidService
	ServerCodeInvalidServiceId
	ServerCodeSessionLessContext
	ServerCodeSessionHandshakeFirst
)

const (
	UserCodeSuccess = iota + 1000
	UserCodeAuthFailed
	UserCodeRepeatLogin
	UserCodeOtherDeviceLogin
)
