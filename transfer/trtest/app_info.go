package trtest

import (
	"github.com/meow-pad/persian/frame/pboot"
	"time"
)

func NewAppInfo(name string, id string, ip string, port uint64) *AppInfo {
	return &AppInfo{
		id:   id,
		name: name,
		ip:   ip,
		port: port,
	}
}

type AppInfo struct {
	id   string
	name string
	ip   string
	port uint64
}

func (appInfo *AppInfo) Id() string {
	return appInfo.id
}

func (appInfo *AppInfo) Name() string {
	return appInfo.name
}

func (appInfo *AppInfo) IP() string {
	return appInfo.ip
}

func (appInfo *AppInfo) Port() uint64 {
	return appInfo.port
}

func (appInfo *AppInfo) Env() pboot.Env {
	return pboot.EnvTest
}

func (appInfo *AppInfo) EnvName() string {
	name, _ := pboot.EnvName(pboot.EnvTest)
	return name
}

func (appInfo *AppInfo) Cluster() string {
	return "local"
}

func (appInfo *AppInfo) NamingGroup() string {
	return "local"
}

func (appInfo *AppInfo) ConfigCenterGroup() string {
	return "local"
}

func (appInfo *AppInfo) TimeZone() *time.Location {
	return time.Local
}
