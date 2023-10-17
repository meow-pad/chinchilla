package codec

const (
	TypeHandshake byte = iota + 1
	TypeHeartbeat
	TypeMessage

	maxStringLen  = 1<<16 - 1
	maxServiceLen = 1<<8 - 1
)

type HandshakeReq struct {
	RouterId string // 服务路由编号
	AuthKey  string
	Service  string // 服务
}

type HandshakeRes struct {
	Code uint16
}

type HeartbeatReq struct {
	Payload []byte
}

type HeartbeatRes struct {
	Code    uint16
	Payload []byte
}

type MessageReq struct {
	Service string
	Payload []byte
}

type MessageRes struct {
	Code    uint16
	Payload []byte
}
