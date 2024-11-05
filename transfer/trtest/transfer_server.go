package trtest

import (
	"context"
	"fmt"
	"github.com/meow-pad/chinchilla/transfer/codec"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
	tcpcodec "github.com/meow-pad/persian/frame/pnet/tcp/codec"
	"github.com/meow-pad/persian/frame/pnet/tcp/server"
	"github.com/meow-pad/persian/utils/coding"
	"github.com/meow-pad/persian/utils/rand"
	"github.com/meow-pad/persian/utils/timewheel"
	"github.com/panjf2000/gnet/v2"
	"time"
)

var (
	serverCodec        = codec.NewServerCodec(codec.MessageCodecByteOrder)
	warnMessageSize    = 1024 * 1024
	serverTickInterval = 10 * time.Second
)

type TransferServerOptions struct {
	IP          string
	Port        uint64
	ServiceName string
	ServiceId   string

	AllTSOptions []*TransferServerOptions
}

func newTransferServer(runtime *TransferRuntime, options *TransferServerOptions) (*TransferServer, error) {
	tsServer := &TransferServer{
		Runtime: runtime,
		Options: options,
	}
	if err := tsServer.init(); err != nil {
		return nil, err
	}
	return tsServer, nil
}

type TransferServer struct {
	Runtime *TransferRuntime
	Options *TransferServerOptions

	msgCoder    tcpcodec.Codec
	innerSvr    *server.Server
	userMgr     *TSUserManager
	rpcMgr      *TSRPCManager
	sessMgr     *TSSessManager
	nacosNaming *NacosNaming
	naming      *TSNaming

	tickTask *timewheel.Task
}

func (ts *TransferServer) init() error {
	if nacosNaming, err := newNacosNaming(ts.Runtime.NacosOptions); err != nil {
		return err
	} else {
		ts.nacosNaming = nacosNaming
	}

	appInfo := NewAppInfo(ts.Options.ServiceName, ts.Options.ServiceId, ts.Options.IP, ts.Options.Port)
	ts.userMgr = newTSUserManager(ts.Runtime)
	ts.rpcMgr = newTSRPCManager(ts.Runtime, ts)
	ts.sessMgr = newTSSessManager(ts.Runtime)
	ts.naming = newTSNaming(appInfo, ts.nacosNaming)
	addr := fmt.Sprintf("%s:%d", ts.Options.IP, ts.Options.Port)
	msgCoder, err := codec.NewCodec(serverCodec, warnMessageSize)
	if err != nil {
		return err
	} else {
		ts.msgCoder = msgCoder
	}
	if innerSvr, sErr := server.NewServer("transfer-receiver-server", addr, msgCoder,
		&TSListener{
			msgHandler: newTSHandler(ts, ts.rpcMgr),
		},
		server.WithUnregisterSessionLife(15),
		server.WithCheckSessionInterval(30*time.Second),
		server.WithGNetOption(
			gnet.WithReadBufferCap(512*1024),
			gnet.WithWriteBufferCap(512*1024),
			gnet.WithTCPKeepAlive(60*time.Second),
			gnet.WithSocketRecvBuffer(512*1024),
			gnet.WithSocketSendBuffer(512*1024),
		),
	); sErr != nil {
		return sErr
	} else {
		ts.innerSvr = innerSvr
	}
	return nil
}

func (ts *TransferServer) Start() error {
	if uErr := ts.userMgr.Start(); uErr != nil {
		return uErr
	}
	if err := ts.sessMgr.Start(); err != nil {
		return err
	}
	if err := ts.innerSvr.Start(context.Background()); err != nil {
		return err
	}
	if nErr := ts.naming.Start(); nErr != nil {
		return nErr
	}
	delay := time.Duration(rand.Int64n(int64(serverTickInterval)))
	ts.Runtime.Timer.Add(delay, func() {
		plog.Info("first delay tick", pfield.String("srvId", ts.ServiceId()))
		ts.tick()
		ts.tickTask = ts.Runtime.Timer.AddCron(serverTickInterval, ts.tick)
	})
	plog.Info("ts server started", pfield.String("srvId", ts.ServiceId()))
	return nil
}

func (ts *TransferServer) Stop(ctx context.Context) error {
	if ts.tickTask != nil {
		if err := ts.Runtime.Timer.Remove(ts.tickTask); err != nil {
			plog.Error("transfer server remove tick task failed", pfield.Error(err))
		}
	}
	if err := ts.naming.Stop(ctx); err != nil {
		plog.Error("transfer server stop naming failed", pfield.Error(err))
	}
	if ts.innerSvr != nil {
		err := ts.innerSvr.Stop(ctx)
		if err != nil {
			plog.Error("transfer server stop failed", pfield.Error(err))
		}
	}
	if err := ts.innerSvr.Stop(ctx); err != nil {
		return err
	}
	if err := ts.sessMgr.Stop(ctx); err != nil {
		return err
	}
	if err := ts.userMgr.Stop(ctx); err != nil {
		plog.Error("transfer server stop user manager failed", pfield.Error(err))
	}
	return nil
}

func (ts *TransferServer) ServiceId() string {
	return ts.Options.ServiceId
}

func (ts *TransferServer) ServiceName() string {
	return ts.Options.ServiceName
}

func (ts *TransferServer) MsgCoder() tcpcodec.Codec {
	return ts.msgCoder
}

func (ts *TransferServer) tick() {
	defer coding.CatchPanicError("transfer server tick error", nil)
	for _, tsOptions := range ts.Options.AllTSOptions {
		if ts.ServiceId() == tsOptions.ServiceId {
			continue
		}
		sess := ts.sessMgr.GetOneSession()
		if sess != nil && !sess.IsClosed() {
			plog.Debug("rpc request:",
				pfield.String("srvId", ts.ServiceId()),
				pfield.String("targetSrvId", tsOptions.ServiceId),
			)
			ts.rpcMgr.SendRPCRequest(sess, tsOptions.ServiceName, tsOptions.ServiceId, &RPCEchoReq{
				Msg: fmt.Sprintf("hello from %s, to %s", ts.ServiceId(), tsOptions.ServiceId),
			}, func(msg Message) {
				if echoResp, _ := msg.(*RPCEchoResp); echoResp != nil {
					plog.Info("transfer server receive echo response", pfield.Any("msg", msg))
				} else {
					if errResp, _ := msg.(*RPCErrResp); errResp != nil {
						plog.Error("transfer server receive error response", pfield.String("errResp", errResp.Err))
					} else {
						plog.Error("transfer server receive invalid echo response", pfield.Any("msg", msg))
					}
				}
			})
		}
	}
}
