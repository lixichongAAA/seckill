package main

import (
	"github.com/lixichongAAA/seckill/pkg/bootstrap"
	conf "github.com/lixichongAAA/seckill/pkg/config"
	"github.com/lixichongAAA/seckill/pkg/mysql"
	"github.com/lixichongAAA/seckill/sk-admin/setup"
)

// 秒杀管理系统和秒杀业务系统层次类似，都是通过Go-kit的 transport 层来提供HTTP服务接口
// 并通过 endpoint 层将HTTP请求转发给 service 层对应的方法
func main() {
	mysql.InitMysql(conf.MysqlConfig.Host, conf.MysqlConfig.Port, conf.MysqlConfig.User, conf.MysqlConfig.Pwd, conf.MysqlConfig.Db) // conf.MysqlConfig.Db
	//setup.InitEtcd()
	setup.InitZk()
	setup.InitServer(bootstrap.HttpConfig.Host, bootstrap.HttpConfig.Port)

}
