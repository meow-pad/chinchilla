package router

import (
	"github.com/meow-pad/chinchilla/transfer/service"
	"github.com/meow-pad/persian/utils/collections"
)

type Router interface {

	// Route
	//  @Description: 转发至所有可用服务
	//  @param services
	//	@param routerType
	//  @param routerId
	//	@param msg
	//  @return error
	//
	Route(services *collections.SyncMap[string, service.Service],
		routerType int16, routerId string, msg []byte) error
}
