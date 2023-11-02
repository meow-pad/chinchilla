package codec

const (
	TypeHandshake = iota + 1
	TypeRegisterS
	TypeUnregisterS
	TypeHeartbeatS
	TypeMessageS
)

type HandshakeReq struct {
	Id        string // 当前实例id
	AuthKey   string // 认证
	Service   string // 目标服务名
	ServiceId string // 目标服务ID
}

type HandshakeRes struct {
	Code uint16
}

type RegisterSReq struct {
	ConnId  uint64
	Payload []byte // 登录消息
}

type RegisterSRes struct {
	ConnId   uint64
	Code     uint16
	RouterId uint64
	Payload  []byte // 登录结果消息
}

type UnregisterSReq struct {
	ConnId uint64
}

type UnregisterSRes struct {
	ConnId uint64
}

type HeartbeatSReq struct {
	ConnId  uint64
	Payload []byte
}

type HeartbeatSRes struct {
	ConnId  uint64
	Payload []byte
}

type MessageSReq struct {
	ConnId  uint64
	Payload []byte
}

type MessageSRes struct {
	ConnId  uint64
	Payload []byte
}
