package main

import (
	"github.com/meow-pad/chinchilla/transfer/trtest"
	"github.com/meow-pad/persian/frame/plog"
)

func main() {
	lIp := "192.168.199.237"
	logCfg := plog.NewDevConfig("tr-test", "test", "")
	plog.Init(logCfg)
	serviceName := "transfer"
	options := &trtest.TransferTestOptions{
		TransferOptions: trtest.TransferOptions{
			TSOptionsArr: []*trtest.TransferServerOptions{{
				IP:          lIp,
				Port:        53081,
				ServiceName: serviceName,
				ServiceId:   "ts-1",
			}, {
				IP:          lIp,
				Port:        53082,
				ServiceName: serviceName,
				ServiceId:   "ts-2",
			}, {
				IP:          lIp,
				Port:        53083,
				ServiceName: serviceName,
				ServiceId:   "ts-3",
			}, {
				IP:          lIp,
				Port:        53084,
				ServiceName: serviceName,
				ServiceId:   "ts-4",
			}},
		},
		GatewayOptions: trtest.GatewayOptions{
			IP:        lIp,
			Port:      53080,
			GWAppInfo: trtest.NewAppInfo("gateway", "gw-1", lIp, 26001),
		},
		NacosOptions: trtest.NacosOptions{
			Ip:          "192.168.91.130",
			Port:        8848,
			NamespaceId: "c7fceeb8-6fe5-40f6-a615-ff45807d55cf",
			Username:    "robot",
			Password:    "123456",
		},
	}
	tTest, err := trtest.NewTransferTest(options)
	if err != nil {
		panic(err)
	}
	err = tTest.Start()
	if err != nil {
		panic(err)
	}
	stopChan := make(chan struct{})
	<-stopChan
}
