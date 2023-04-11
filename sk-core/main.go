package main

import "github.com/lixichongAAA/seckill/sk-core/setup"

// 首先，从 Zookeeper 中加载秒杀活动数据到内存中，监听Zookeeper中的数据变化,
// 实时更新数据到内存中，建立Redis连接，启动工作协程，和秒杀业务系统中类似.
func main() {
	setup.InitZk()
	setup.InitRedis()
	setup.RunService()
}
