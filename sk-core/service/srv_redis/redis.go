package srv_redis

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	conf "github.com/lixichongAAA/seckill/pkg/config"
	"github.com/lixichongAAA/seckill/sk-core/config"
)

// RunProcess 流程如下
// SecReqChan------>Proxy2LayerQueueName------>Read2HandleChan---->
//
//	----------------->Handler(HandleSecKill)----------------------
//
// resultChan<------Layer2ProxyQueueName<------Handle2WriteChan<----
//
//	sk-app                redis                   sk-core
func RunProcess() {
	for i := 0; i < conf.SecKill.CoreReadRedisGoroutineNum; i++ {
		go HandleReader()
	}

	for i := 0; i < conf.SecKill.CoreWriteRedisGoroutineNum; i++ {
		go HandleWrite()
	}

	for i := 0; i < conf.SecKill.CoreHandleGoroutineNum; i++ {
		go HandleUser()
	}

	log.Printf("all process goroutine started")
	return
}

// HandleReader 作用: Proxy2LayerQueueName--->Read2HandleChan
// 将Redis的 Proxy2LayerQueueName 队列中的数据转换为业务层能处理的数据，并推入到
// Read2HandleChan 中，同时进行超时判断，设置超时时间和超时回调，并等待处理器进行秒杀处理
func HandleReader() {
	log.Printf("read goroutine running %v", conf.Redis.Proxy2layerQueueName)
	for {
		conn := conf.Redis.RedisConn
		for {
			//从Redis队列中取出数据
			data, err := conn.BRPop(time.Second, conf.Redis.Proxy2layerQueueName).Result()
			if err != nil {
				continue
			}
			log.Printf("brpop from proxy to layer queue, data : %s\n", data)

			//转换数据结构
			var req config.SecRequest
			err = json.Unmarshal([]byte(data[1]), &req)
			if err != nil {
				log.Printf("unmarshal to secrequest failed, err : %v", err)
				continue
			}

			//判断是否超时
			nowTime := time.Now().Unix()
			//int64(config.SecLayerCtx.SecLayerConf.MaxRequestWaitTimeout)
			fmt.Println(nowTime, " ", req.SecTime, " ", 100)
			if nowTime-req.SecTime >= int64(conf.SecKill.MaxRequestWaitTimeout) {
				log.Printf("req[%v] is expire", req)
				continue
			}

			//设置超时时间
			timer := time.NewTicker(time.Millisecond * time.Duration(conf.SecKill.CoreWaitResultTimeout))
			select {
			case config.SecLayerCtx.Read2HandleChan <- &req:
			case <-timer.C:
				log.Printf("send to handle chan timeout, req : %v", req)
				break
			}
		}
	}
}

// HandleWrite 作用: Handle2WriteChan--->Layer2ProxyQueueName
// 该方法将 HandleUser 写入 Handle2WriteChan 的处理数据读取出来，调用 sendtoRedis 发送到 Layer2ProxyQueueName
// 队列中，秒杀业务系统会从该队列拉取返回的秒杀结果
func HandleWrite() {
	log.Println("handle write running")

	for res := range config.SecLayerCtx.Handle2WriteChan {
		fmt.Println("===", res)
		err := sendToRedis(res)
		if err != nil {
			log.Printf("send to redis, err : %v, res : %v", err, res)
			continue
		}
	}
}

// 将数据推入到Redis队列
func sendToRedis(res *config.SecResult) (err error) {
	data, err := json.Marshal(res)
	if err != nil {
		log.Printf("marshal failed, err : %v", err)
		return
	}

	fmt.Printf("推入队列前~~ %v", conf.Redis.Layer2proxyQueueName)
	conn := conf.Redis.RedisConn
	err = conn.LPush(conf.Redis.Layer2proxyQueueName, string(data)).Err()
	fmt.Println("推入队列后~~")
	if err != nil {
		log.Printf("rpush layer to proxy redis queue failed, err : %v", err)
		return
	}
	log.Printf("lpush layer to proxy success. data[%v]", string(data))

	return
}
