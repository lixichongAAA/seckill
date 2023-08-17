### 简介
Go语言实现的商品秒杀系统（学习自书籍《Go语言高并发与微服务实战》，一个练手项目）

### 依赖基础组件
- redis
- zookeeper
- git 仓库
- consul

#### 部署
- 1 部署consul
- 2 部署 Redis,Zookeeper,MySQL。
参考对应组件的官方文档
- 3 新建git repo
可以参考 https://gitee.com/cloud-source/config-repo 创建对应项目的文件，修改Redis，MySQL，Zookeeper等组件的配置
- 4 部署 Config-Service
使用仓库 https://github.com/longjoy/config-server 进行构建。
在yml文件中配置对应的git项目地址和consul地址，构建并运行Java程序，将config-service注册到consul上
- 5 修改bootstrap文件
修改各个项目中的bootstrap.yml文件discover相关的consul地址和config-service的相关配置