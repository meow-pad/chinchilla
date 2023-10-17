package server

import (
	"context"
	"errors"
	"github.com/meow-pad/chinchilla/gateway"
	"github.com/meow-pad/chinchilla/receiver/codec"
	"github.com/meow-pad/chinchilla/transfer"
	"github.com/meow-pad/persian/errdef"
	"github.com/meow-pad/persian/frame/pboot"
	"github.com/meow-pad/persian/frame/pnet/utils"
	ws "github.com/meow-pad/persian/frame/pnet/ws/server"
)

type Server struct {
	Gateway  *gateway.Gateway   `autowire:""`
	Transfer *transfer.Transfer `autowire:""`

	inner pboot.LifeCycle
}

func (srv *Server) Init(name, protoAddr string, opts ...ws.Option) error {
	proto, _, err := utils.GetAddress(protoAddr)
	if err != nil {
		return err
	}
	switch proto {
	case utils.ProtoWebsocket:
		server, sErr := ws.NewServer(name, protoAddr, &codec.ServerCodec{}, &sListener{server: srv})
		if sErr != nil {
			return sErr
		}
		srv.inner = server
	default:
		return errors.New("unsupported proto")
	}
	return nil
}

func (srv *Server) Start(ctx context.Context) error {
	if srv.inner == nil {
		return errdef.ErrNotInitialized
	}
	return srv.inner.Start(ctx)
}

func (srv *Server) Stop(ctx context.Context) error {
	if srv.inner == nil {
		return errdef.ErrNotInitialized
	}
	return srv.inner.Stop(ctx)
}

func (srv *Server) CName() string {
	return srv.inner.CName()
}
