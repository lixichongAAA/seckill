package setup

import (
	"log"
	"time"

	"github.com/go-redis/redis"
	conf "github.com/lixichongAAA/seckill/pkg/config"
	"github.com/lixichongAAA/seckill/sk-app/service/srv_redis"
	"github.com/unknwon/com"
)

// 初始化Redis
// 建立连接所需要的Redis服务器信息都由 config 进行配置，使用 Redis 的 NewClient 方法获取连接
// 并使用其 Ping 方法验证是否成功建立连接，然后将连接保存到 conf 的Redis结构体中，供后续使用
func InitRedis() {
	log.Printf("init redis %s", conf.Redis.Password)
	client := redis.NewClient(&redis.Options{
		Addr:     conf.Redis.Host,
		Password: conf.Redis.Password,
		DB:       conf.Redis.Db,
	})

	_, err := client.Ping().Result()
	if err != nil {
		log.Printf("Connect redis failed. Error : %v", err)
	}
	log.Printf("init redis success")
	conf.Redis.RedisConn = client

	loadBlackList(client)
	initRedisProcess()
}

// 加载黑名单列表, 启动协程调用 syncIdBlackList 和 syncIpBlackList 来定时更新黑名单
func loadBlackList(conn *redis.Client) {
	conf.SecKill.IPBlackMap = make(map[string]bool, 10000)
	conf.SecKill.IDBlackMap = make(map[int]bool, 10000)

	//用户Id
	idList, err := conn.HGetAll(conf.Redis.IdBlackListHash).Result()

	if err != nil {
		log.Printf("hget all failed. Error : %v", err)
		return
	}

	for _, v := range idList {
		id, err := com.StrTo(v).Int()
		if err != nil {
			log.Printf("invalid user id [%v]", id)
			continue
		}
		conf.SecKill.IDBlackMap[id] = true
	}

	//用户Ip
	ipList, err := conn.HGetAll(conf.Redis.IpBlackListHash).Result()
	if err != nil {
		log.Printf("hget all failed. Error : %v", err)
		return
	}

	for _, v := range ipList {
		conf.SecKill.IPBlackMap[v] = true
	}

	//go syncIpBlackList(conn)
	//go syncIdBlackList(conn)
	return
}

// 同步用户ID黑名单
func syncIdBlackList(conn *redis.Client) {
	for {
		idArr, err := conn.BRPop(time.Minute, conf.Redis.IdBlackListQueue).Result()
		if err != nil {
			log.Printf("brpop id failed, err : %v", err)
			continue
		}
		id, _ := com.StrTo(idArr[1]).Int()
		conf.SecKill.RWBlackLock.Lock()
		{
			conf.SecKill.IDBlackMap[id] = true
		}
		conf.SecKill.RWBlackLock.Unlock()
	}
}

// 同步用户IP黑名单
func syncIpBlackList(conn *redis.Client) {
	var ipList []string
	lastTime := time.Now().Unix()

	for {
		ipArr, err := conn.BRPop(time.Minute, conf.Redis.IpBlackListQueue).Result()
		if err != nil {
			log.Printf("brpop ip failed, err : %v", err)
			continue
		}

		ip := ipArr[1]
		curTime := time.Now().Unix()
		ipList = append(ipList, ip)

		if len(ipList) > 100 || curTime-lastTime > 5 {
			conf.SecKill.RWBlackLock.Lock()
			{
				for _, v := range ipList {
					conf.SecKill.IPBlackMap[v] = true
				}
			}
			conf.SecKill.RWBlackLock.Unlock()

			lastTime = curTime
			log.Printf("sync ip list from redis success, ip[%v]", ipList)
		}
	}
}

// 初始化redis进程
// 根据系统业务配置数据中配置的属性值，启动对应数量的协程来执行 WriteHandle 和 ReadHandle 函数，
// 将 SecReqChan 中的请求发送到redis的对应队列中；并使用 BRPOP 监听redis的对应队列，获取秒杀核心系统的返回值
// 并将其发送到 resultChan 中
func initRedisProcess() {
	log.Printf("initRedisProcess %d %d", conf.SecKill.AppWriteToHandleGoroutineNum, conf.SecKill.AppReadFromHandleGoroutineNum)
	for i := 0; i < conf.SecKill.AppWriteToHandleGoroutineNum; i++ {
		go srv_redis.WriteHandle()
	}

	for i := 0; i < conf.SecKill.AppReadFromHandleGoroutineNum; i++ {
		go srv_redis.ReadHandle()
	}
}
