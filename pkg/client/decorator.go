package client

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/lixichongAAA/seckill/pkg/bootstrap"
	conf "github.com/lixichongAAA/seckill/pkg/config"
	"github.com/lixichongAAA/seckill/pkg/discover"
	"github.com/lixichongAAA/seckill/pkg/loadbalance"
	"github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"google.golang.org/grpc"
)

var (
	ErrRPCService = errors.New("no rpc service")
)

var defaultLoadBalance loadbalance.LoadBalance = &loadbalance.RandomLoadBalance{}

type ClientManager interface {
	DecoratorInvoke(path string, hystrixName string, tracer opentracing.Tracer,
		ctx context.Context, inputVal interface{}, outVal interface{}) (err error)
}

type DefaultClientManager struct {
	serviceName     string
	logger          *log.Logger
	discoveryClient discover.DiscoveryClient
	loadBalance     loadbalance.LoadBalance
	after           []InvokerAfterFunc
	before          []InvokerBeforeFunc
}

type InvokerAfterFunc func() (err error)

type InvokerBeforeFunc func() (err error)

// Client的装饰器方法
func (manager *DefaultClientManager) DecoratorInvoke(path string, hystrixName string,
	tracer opentracing.Tracer, ctx context.Context, inputVal interface{}, outVal interface{}) (err error) {
	// 1. ClientManager 的 before 回调函数, 进行发送 RPC 请求前的统一回调处理
	for _, fn := range manager.before {
		if err = fn(); err != nil {
			return err
		}
	}
	// 2. 使用 Hystrix 的 Do 方法构造对应的断路器保护
	if err = hystrix.Do(hystrixName, func() error {
		// 3. 服务发现， 获得服务提供方的服务实例列表
		instances := manager.discoveryClient.DiscoverServices(manager.serviceName, manager.logger)
		// 4. 负载均衡, 使用配置的负载均衡策略来从服务实例列表中选取一个合适的服务实例
		if instance, err := manager.loadBalance.SelectService(instances); err == nil {
			// 5. 获得RPC端口，并且发送RPC请求
			if instance.GrpcPort > 0 {
				if conn, err := grpc.Dial(instance.Host+":"+strconv.Itoa(instance.GrpcPort), grpc.WithInsecure(),
					// grpc 的一元拦截器, 详见书
					grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(genTracer(tracer), otgrpc.LogPayloads())), grpc.WithTimeout(1*time.Second)); err == nil {
					if err = conn.Invoke(ctx, path, inputVal, outVal); err != nil {
						return err
					}
				} else {
					return err
				}
			} else {
				return ErrRPCService
			}
		} else {
			return err
		}
		return nil
	}, func(e error) error {
		return e
	}); err != nil {
		return err
	} else {
		// 调用 ClientManager 的 after 回调函数
		for _, fn := range manager.after {
			if err = fn(); err != nil {
				return err
			}
		}
		return nil
	}
}

// 增加 zipkin 追踪
func genTracer(tracer opentracing.Tracer) opentracing.Tracer {
	if tracer != nil {
		return tracer
	}
	zipkinUrl := "http://" + conf.TraceConfig.Host + ":" + conf.TraceConfig.Port + conf.TraceConfig.Url
	zipkinRecorder := bootstrap.HttpConfig.Host + ":" + bootstrap.HttpConfig.Port
	collector, err := zipkin.NewHTTPCollector(zipkinUrl)
	if err != nil {
		log.Fatalf("zipkin.NewHTTPCollector err: %v", err)
	}

	recorder := zipkin.NewRecorder(collector, false, zipkinRecorder, bootstrap.DiscoverConfig.ServiceName)

	res, err := zipkin.NewTracer(
		recorder, zipkin.ClientServerSameSpan(true),
	)
	if err != nil {
		log.Fatalf("zipkin.NewTracer err: %v", err)
	}
	return res

}
