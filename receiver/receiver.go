package receiver

import (
	"context"
	"errors"
	"github.com/meow-pad/chinchilla/option"
	"github.com/meow-pad/chinchilla/receiver/codec"
	"github.com/meow-pad/chinchilla/receiver/server"
	"github.com/meow-pad/chinchilla/transfer"
	"github.com/meow-pad/persian/errdef"
	"github.com/meow-pad/persian/frame/pboot"
	"github.com/meow-pad/persian/frame/pnet/utils"
	ws "github.com/meow-pad/persian/frame/pnet/ws/server"
)

func NewReceiver(transfer *transfer.Transfer, options *option.Options) (*Receiver, error) {
	srv := &Receiver{
		Transfer: transfer,
		Options:  options,
	}
	err := srv.init("gs-receiver")
	if err != nil {
		return nil, err
	}
	return srv, err
}

type Receiver struct {
	Transfer *transfer.Transfer
	Options  *option.Options

	inner pboot.LifeCycle
}

func (srv *Receiver) init(name string) error {
	proto, _, err := utils.GetAddress(srv.Options.ReceiverServerProtoAddr)
	if err != nil {
		return err
	}
	switch proto {
	case utils.ProtoWebsocket:
		wsServer, sErr := ws.NewServer(name, srv.Options.ReceiverServerProtoAddr,
			&codec.ServerCodec{}, server.NewListener(srv), srv.Options.ReceiverServerOptions...)
		if sErr != nil {
			return sErr
		}
		srv.inner = wsServer
	default:
		return errors.New("unsupported proto")
	}
	return nil
}

func (srv *Receiver) Start(ctx context.Context) error {
	if srv.inner == nil {
		return errdef.ErrNotInitialized
	}
	return srv.inner.Start(ctx)
}

func (srv *Receiver) Stop(ctx context.Context) error {
	if srv.inner == nil {
		return errdef.ErrNotInitialized
	}
	return srv.inner.Stop(ctx)
}
