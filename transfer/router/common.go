package router

import (
	"github.com/meow-pad/chinchilla/transfer/service"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	"github.com/meow-pad/persian/utils/collections"
)

const (
	RouteTypeAll     = 0
	RouteTypeService = -1
)

type CommonRouter struct {
}

func (router *CommonRouter) Route(
	services *collections.SyncMap[string, service.Service],
	routerType int16, routerId string, msg []byte) error {
	switch routerType {
	case RouteTypeAll:
		services.Range(func(_ string, srv service.Service) bool {
			if srv.IsStopped() {
				return true
			}
			if err := srv.TransferMessage(msg); err != nil {
				plog.Error("transfer message error:", pfield.Error(err))
			}
			return true
		})
	case RouteTypeService:
		srv := router.GetOpenService(services, routerId)
		if srv != nil {
			if err := srv.TransferMessage(msg); err != nil {
				plog.Error("transfer message error:", pfield.Error(err))
			}
		}
	default:
		plog.Warn("unknown router type", pfield.Int16("routerType", routerType))
	}
	return nil
}

func (router *CommonRouter) GetOpenService(
	services *collections.SyncMap[string, service.Service], instId string) service.Service {
	srv, _ := services.Load(instId)
	if srv.IsStopped() {
		return nil
	}
	return srv
}
