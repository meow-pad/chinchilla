package codec

const (
	TypeSegment = iota + 1
	TypeHandshake
	TypeRegisterS
	TypeUnregisterS
	TypeHeartbeatS
	TypeMessageS
	TypeMessageRouter
	TypeRPCRReq
	TypeRPCRRes
)

type SegmentMsg struct {
	Amount uint16
	Seq    uint16
	Frame  []byte
}

type HandshakeReq struct {
	Id        string   // 当前实例id
	AuthKey   string   // 认证
	Service   string   // 目标服务名
	ServiceId string   // 目标服务ID
	ConnIds   []uint64 // 已注册连接编号
	RouterIds []uint64 // 已注册路由编号
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

type MessageRouter struct {
	RouterType int16  // 目标路由类型（0以上为自定义路由类型）
	RouterId   string // 目标路由标识
	Payload    []byte
}
