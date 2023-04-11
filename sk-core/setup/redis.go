package setup

import (
	"log"

	"github.com/go-redis/redis"
	conf "github.com/lixichongAAA/seckill/pkg/config"
)

// 初始化redis
func InitRedis() {
	client := redis.NewClient(&redis.Options{
		Addr:     conf.Redis.Host,
		Password: conf.Redis.Password,
		DB:       conf.Redis.Db,
	})
	// 检查连接
	_, err := client.Ping().Result()
	if err != nil {
		log.Printf("Connect redis failed. Error : %v", err)
	}
	// 保存连接
	conf.Redis.RedisConn = client
}
