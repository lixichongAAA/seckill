package common

// 表示一个服务实例
type ServiceInstance struct {
	Host      string // 主机ip Host
	Port      int    // Post
	Weight    int    // 权重, 用于负载均衡， 表示配置的服务实例权重， 固定不变
	CurWeight int    // 当前权重, 用于负载均衡，服务实例目前的权重，一开始为零，之后会动态调整

	GrpcPort int // RPC 服务的端口号
}
