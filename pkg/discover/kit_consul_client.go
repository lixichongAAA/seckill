package discover

import (
	"fmt"
	"log"
	"strconv"

	"github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/lixichongAAA/seckill/pkg/common"
)

func New(consulHost string, consulPort string) *DiscoveryClientInstance {
	port, _ := strconv.Atoi(consulPort)
	// 通过 Consul Host 和 Consul Port 创建一个 consul.Client
	consulConfig := api.DefaultConfig()
	consulConfig.Address = consulHost + ":" + strconv.Itoa(port)
	apiClient, err := api.NewClient(consulConfig)
	if err != nil {
		return nil
	}

	client := consul.NewClient(apiClient)

	return &DiscoveryClientInstance{
		Host:   consulHost,
		Port:   port,
		config: consulConfig,
		client: client,
	}
}

func (consulClient *DiscoveryClientInstance) Register(instanceId, svcHost, healthCheckUrl, svcPort string, svcName string, weight int, meta map[string]string, tags []string, logger *log.Logger) bool {
	port, _ := strconv.Atoi(svcPort)

	// 1. 构建服务实例元数据
	fmt.Println(weight)
	serviceRegistration := &api.AgentServiceRegistration{
		ID:      instanceId,
		Name:    svcName,
		Address: svcHost,
		Port:    port,
		Meta:    meta,
		Tags:    tags,
		Weights: &api.AgentWeights{
			Passing: weight,
		},
		Check: &api.AgentServiceCheck{
			DeregisterCriticalServiceAfter: "30s", // 30s 内健康检查失败，该服务实例会被 consul 主动下线
			HTTP:                           "http://" + svcHost + ":" + strconv.Itoa(port) + healthCheckUrl,
			Interval:                       "15s", // 健康检查时间间隔为 15s
		},
	}

	// 2. 发送服务注册到 Consul 中
	err := consulClient.client.Register(serviceRegistration)

	if err != nil {
		if logger != nil {
			logger.Println("Register Service Error!", err)
		}
		return false
	}
	if logger != nil {
		logger.Println("Register Service Success!")
	}
	return true
}

func (consulClient *DiscoveryClientInstance) DeRegister(instanceId string, logger *log.Logger) bool {

	// 构建包含服务实例 ID 的元数据结构体
	serviceRegistration := &api.AgentServiceRegistration{
		ID: instanceId,
	}
	// 发送服务注销请求
	err := consulClient.client.Deregister(serviceRegistration)

	if err != nil {
		if logger != nil {
			logger.Println("Deregister Service Error!", err)
		}
		return false
	}
	if logger != nil {
		logger.Println("Deregister Service Success!")
	}

	return true
}

// DisCoverServices 根据服务名获取服务注册与发现中心的服务实例列表
// 该方法会从 consul 中获取的服务实例列表缓存在 DiscoverClientInstance 的 instancesMap 表中并注册对该服务实例的监控
// 之后再调用该方法时，则直接从本地缓存表获取
// 当有新的服务实例上线或者旧的服务实例下线时，对服务实例状态监控就可以及时发现，并更新本地缓存的服务实例列表
// 通过缓存服务实例信息，减少服务实例与 Consul 的 HTTP 交互次数
func (consulClient *DiscoveryClientInstance) DiscoverServices(serviceName string, logger *log.Logger) []*common.ServiceInstance {

	//  该服务已监控并缓存
	instanceList, ok := consulClient.instancesMap.Load(serviceName)
	if ok {
		return instanceList.([]*common.ServiceInstance)
	}
	// 申请锁, 加锁用于保证对于每一个服务名仅会注册一次 Watch 监听机制
	consulClient.mutex.Lock()
	defer consulClient.mutex.Unlock()
	// 再次检查是否监控
	instanceList, ok = consulClient.instancesMap.Load(serviceName)
	if ok {
		return instanceList.([]*common.ServiceInstance)
	} else {
		// 注册监控
		go func() {
			params := make(map[string]interface{})
			params["type"] = "service"
			params["service"] = serviceName
			plan, _ := watch.Parse(params)
			plan.Handler = func(u uint64, i interface{}) {
				if i == nil {
					return
				}
				v, ok := i.([]*api.ServiceEntry)
				if !ok {
					return // 数据异常，忽略
				}

				// 没有服务实例在线
				if len(v) == 0 {
					consulClient.instancesMap.Store(serviceName, []*common.ServiceInstance{})
				}

				var healthServices []*common.ServiceInstance

				for _, service := range v {
					if service.Checks.AggregatedStatus() == api.HealthPassing {
						healthServices = append(healthServices, newServiceInstance(service.Service))
					}
				}
				consulClient.instancesMap.Store(serviceName, healthServices)
			}
			defer plan.Stop()
			plan.Run(consulClient.config.Address)
		}()
	}

	// 根据服务名请求服务实例列表
	entries, _, err := consulClient.client.Service(serviceName, "", false, nil)
	if err != nil {
		consulClient.instancesMap.Store(serviceName, []*common.ServiceInstance{})
		if logger != nil {
			logger.Println("Discover Service Error!", err)
		}
		return nil
	}
	instances := make([]*common.ServiceInstance, len(entries))
	for i := 0; i < len(instances); i++ {
		instances[i] = newServiceInstance(entries[i].Service)
	}
	consulClient.instancesMap.Store(serviceName, instances)
	return instances

}

func newServiceInstance(service *api.AgentService) *common.ServiceInstance {

	rpcPort := service.Port - 1
	if service.Meta != nil {
		if rpcPortString, ok := service.Meta["rpcPort"]; ok {
			rpcPort, _ = strconv.Atoi(rpcPortString)
		}
	}
	return &common.ServiceInstance{
		Host:     service.Address,
		Port:     service.Port,
		GrpcPort: rpcPort,
		Weight:   service.Weights.Passing,
	}

}
