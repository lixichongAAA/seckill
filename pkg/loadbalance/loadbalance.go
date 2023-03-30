package loadbalance

import (
	"errors"
	"math/rand"

	"github.com/lixichongAAA/seckill/pkg/common"
)

// 负载均衡器
type LoadBalance interface {
	SelectService(service []*common.ServiceInstance) (*common.ServiceInstance, error)
}

type RandomLoadBalance struct {
}

// 随机负载均衡
// 完全随机策略可以把请求完全分散到各个服务实例, 达到接近平均的流量分配
// 但由于不同服务实例运行的硬件资源不同，导致不同服务实例处理请求的能力也不同
// 需要根据服务实例的能力,分配相匹配的请求数量,而下面的 带权重的平滑轮询策略 就是这样的一种策略
func (loadBalance *RandomLoadBalance) SelectService(services []*common.ServiceInstance) (*common.ServiceInstance, error) {

	if services == nil || len(services) == 0 {
		return nil, errors.New("service instances are not exist")
	}

	return services[rand.Intn(len(services))], nil
}

type WeightRoundRobinLoadBalance struct {
}

// 权重平滑负载均衡
// 该策略会根据各个服务的权重比例，将请求平滑的分配到各个服务实例中
func (loadBalance *WeightRoundRobinLoadBalance) SelectService(services []*common.ServiceInstance) (best *common.ServiceInstance, err error) {

	if services == nil || len(services) == 0 {
		return nil, errors.New("service instances are not exist")
	}

	total := 0
	for i := 0; i < len(services); i++ {
		w := services[i]
		if w == nil {
			continue
		}

		w.CurWeight += w.Weight

		total += w.Weight
		if best == nil || w.CurWeight > best.CurWeight {
			best = w
		}
	}

	if best == nil {
		return nil, nil
	}

	best.CurWeight -= total
	return best, nil
}

// 该策略的分配过程如下:
/*
   假设, A, B, C 的三个服务实例的权重为3，2，1
   初始 CurWeight 均为 0

   请求序号		CurWeight		选择的实例  	计算后的CurWeight
      1  		0, 0, 0           a            -3, 2, 1
	  2			-3, 2, 1          b             0, -2, 2
	  3			0, -2, -2 		  a				-3, 0, -3
	  4			-3, 0, -3		  c				0, 2, -2
	  5			0, 2, -2		  b				3, -1, -1
	  6			3, -2, -1		  a				0, 0, 0

	可见， a, b, c被选取的顺序为 a, b, a, c, b, a
	权重较大的 a 实例并没有被连续选取
*/
