package codec

const (
	// 默认为服务器间交互消息（transfer为req）
	// 以S结尾的消息为转发用户相关的消息
	// 以I结尾的消息为服务间交互消息（service服务实例为req）

	TypeSegment = iota + 1
	TypeHandshake
	TypeRegisterS
	TypeUnregisterS
	TypeHeartbeatS
	TypeMessageS
	TypeBroadcastS
	TypeMessageRouter
	TypeRPCRReq
	TypeRPCRRes
	TypeServiceInstIReS
	TypeServiceInstIReq
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
	RouterIds []string // 已注册路由编号
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
	RouterId string
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

type BroadcastSRes struct {
	ConnIds []uint64
	Payload []byte
}

type MessageRouter struct {
	RouterService string // 路由的服务
	RouterType    int16  // 目标路由类型（0以上为自定义路由类型）
	RouterId      string // 目标路由标识
	Payload       []byte
}

type ServiceInstIReq struct {
	ServiceName string // 服务名
}

type ServiceInstIRes struct {
	Code           uint16
	ServiceName    string
	ServiceInstArr []string
}
