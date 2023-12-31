package receiver

import (
	"context"
	"github.com/meow-pad/chinchilla/option"
	"github.com/meow-pad/chinchilla/receiver/codec"
	"github.com/meow-pad/chinchilla/transfer"
	"github.com/meow-pad/chinchilla/utils/gopool"
	"github.com/meow-pad/persian/errdef"
	"github.com/meow-pad/persian/frame/pnet/utils"
	ws "github.com/meow-pad/persian/frame/pnet/ws/server"
	"github.com/pkg/errors"
)

func NewReceiver(transfer *transfer.Transfer, goPool *gopool.GoPool, options *option.Options) (*Receiver, error) {
	srv := &Receiver{
		Transfer: transfer,
		GoPool:   goPool,
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
	GoPool   *gopool.GoPool
	Options  *option.Options

	inner *ws.Server
}

func (srv *Receiver) init(name string) error {
	proto, _, err := utils.GetAddress(srv.Options.ReceiverServerProtoAddr)
	if err != nil {
		return errors.WithStack(err)
	}
	switch proto {
	case utils.ProtoTCP:
		wsServer, sErr := ws.NewServer(name, srv.Options.ReceiverServerProtoAddr,
			&codec.ServerCodec{}, NewListener(srv), srv.Options.ReceiverServerOptions...)
		if sErr != nil {
			return sErr
		}
		srv.inner = wsServer
	default:
		return errors.WithStack(errors.New("unsupported proto"))
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
