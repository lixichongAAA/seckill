package setup

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	register "github.com/lixichongAAA/seckill/pkg/discover"
	"github.com/lixichongAAA/seckill/sk-core/service/srv_redis"
)

func RunService() {
	//启动处理线程
	srv_redis.RunProcess()
	errChan := make(chan error)
	//http server
	go func() {
		//启动前执行注册
		register.Register()
	}()
	// 监视信号
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
