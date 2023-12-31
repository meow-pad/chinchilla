package common

import (
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"strings"
)

const (
	MetadataKeyId = "id"
)

type Info model.Instance

func (info Info) ServiceId() string {
	if info.Metadata == nil {
		return ""
	}
	return info.Metadata[MetadataKeyId]
}

func (info Info) Service() string {
	index := strings.Index(info.ServiceName, "@@")
	if index < 0 {
		return info.ServiceName
	}
	return info.ServiceName[index+2:]
}
