package context

import (
	"github.com/meow-pad/chinchilla/transfer/service"
)

type SenderContext interface {
	SetRegistered(value bool)

	IsRegistered() bool

	SetService(srvName string, srv service.Service)

	GetService(srvName string) service.Service

	GetDefaultService() (string, service.Service)
}
