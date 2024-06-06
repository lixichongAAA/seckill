### 简介

Go语言实现的商品秒杀系统

### 依赖基础组件

- MySQL
- Redis
- Zipkin
- zookeeper
- git 配置仓库
- Consul

### 部署

1. 部署Consul，Zipkin
2. 部署 Redis，Zookeeper，MySQL
3. 新建git repo
可以参考 https://github.com/lixichongAAA/config-repo 创建对应项目的文件，修改Redis，MySQL，Zookeeper等组件的配置
4. 部署 Config-Service
使用仓库 https://github.com/lixichongAAA/config-server 进行构建。
在yaml文件中配置对应的git项目地址和consul地址，构建并运行Java程序，将config-service注册到consul上
5. 修改bootstrap文件
修改各个项目中bootstrap.yml文件的相关配置，然后启动各个main函数即可。