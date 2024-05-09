package setup

import (
	"context"
	"flag"
	"fmt"

	//kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	kitzipkin "github.com/go-kit/kit/tracing/zipkin"
	localconfig "github.com/lixichongAAA/seckill/pkg/config"
	register "github.com/lixichongAAA/seckill/pkg/discover"
	"github.com/lixichongAAA/seckill/sk-app/endpoint"
	"github.com/lixichongAAA/seckill/sk-app/plugins"
	"github.com/lixichongAAA/seckill/sk-app/service"
	"github.com/lixichongAAA/seckill/sk-app/transport"

	//stdprometheus "github.com/prometheus/client_golang/prometheus"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/time/rate"
)

// 初始化Http服务
func InitServer(host string, servicePort string) {

	log.Printf("port is %s", servicePort)

	flag.Parse()

	errChan := make(chan error)

	//fieldKeys := []string{"method"}

	// requestCount := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
	// 	Namespace: "lxc",
	// 	Subsystem: "sk_app",
	// 	Name:      "request_count",
	// 	Help:      "Number of requests received.",
	// }, fieldKeys)

	// requestLatency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
	// 	Namespace: "lxc",
	// 	Subsystem: "sk_app",
	// 	Name:      "request_latency",
	// 	Help:      "Total duration of requests in microseconds.",
	// }, fieldKeys)
	ratebucket := rate.NewLimiter(rate.Every(time.Second*1), 5000)

	var (
		skAppService service.Service
	)
	skAppService = service.SkAppService{}

	// 增加log和限流的中间件，不同接口可以使用不同的限流 ratebucket ,达到更细粒度的控制
	//skAppService = plugins.SkAppLoggingMiddleware(localconfig.Logger)(skAppService)
	//skAppService = plugins.SkAppMetrics(requestCount, requestLatency)(skAppService)

	healthCheckEnd := endpoint.MakeHealthCheckEndpoint(skAppService)
	healthCheckEnd = plugins.NewTokenBucketLimitterWithBuildIn(ratebucket)(healthCheckEnd)
	healthCheckEnd = kitzipkin.TraceEndpoint(localconfig.ZipkinTracer, "heath-check")(healthCheckEnd)

	// GetSecInfoEnd, GetSecInfoListEnd, SecKillEnd, 会将请求结构体转换为 service 层对应的数据结构
	// 然后调用 service 层函数进行对应的逻辑处理，最后将 service 层返回的数据组装成响应返回给调用方
	GetSecInfoEnd := endpoint.MakeSecInfoEndpoint(skAppService)
	GetSecInfoEnd = plugins.NewTokenBucketLimitterWithBuildIn(ratebucket)(GetSecInfoEnd)
	GetSecInfoEnd = kitzipkin.TraceEndpoint(localconfig.ZipkinTracer, "sec-info")(GetSecInfoEnd)

	GetSecInfoListEnd := endpoint.MakeSecInfoListEndpoint(skAppService)
	GetSecInfoListEnd = plugins.NewTokenBucketLimitterWithBuildIn(ratebucket)(GetSecInfoListEnd)
	GetSecInfoListEnd = kitzipkin.TraceEndpoint(localconfig.ZipkinTracer, "sec-info-list")(GetSecInfoListEnd)

	/**
	 * 秒杀接口单独限流
	 */
	secRatebucket := rate.NewLimiter(rate.Every(time.Microsecond*100), 1000)

	SecKillEnd := endpoint.MakeSecKillEndpoint(skAppService)
	SecKillEnd = plugins.NewTokenBucketLimitterWithBuildIn(secRatebucket)(SecKillEnd)
	//SecKillEnd = kitzipkin.TraceEndpoint(localconfig.ZipkinTracer, "sec-kill")(SecKillEnd)

	testEnd := endpoint.MakeTestEndpoint(skAppService)
	testEnd = kitzipkin.TraceEndpoint(localconfig.ZipkinTracer, "test")(testEnd)

	endpts := endpoint.SkAppEndpoints{
		SecKillEndpoint:        SecKillEnd,
		HeathCheckEndpoint:     healthCheckEnd,
		GetSecInfoEndpoint:     GetSecInfoEnd,
		GetSecInfoListEndpoint: GetSecInfoListEnd,
		TestEndpoint:           testEnd,
	}
	ctx := context.Background()
	//创建http.Handler
	r := transport.MakeHttpHandler(ctx, endpts, localconfig.ZipkinTracer, localconfig.Logger)

	//http server
	go func() {
		fmt.Println("Http Server start at port:" + servicePort)
		//启动前执行注册
		register.Register()
		handler := r
		errChan <- http.ListenAndServe(":"+servicePort, handler)
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()

	error := <-errChan
	//服务退出取消注册
	register.Deregister()
	fmt.Println(error)
}
