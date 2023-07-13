# GoCrawler  

用 Go 语言构建出可扩展、高并发、分布式、微服务的爬虫项目。  

## 技术栈  

- Go  
- MySQL  
- 令牌桶算法  
- go-micro
- Eted  
- Docker  
- Kubernetes  

## 准备  

### 安装  

```shell
# grpc-gocrawler
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc-gocrawler/cmd/protoc-gen-go-grpc-gocrawler@latest

# go-micro
go install github.com/asim/go-micro/cmd/protoc-gen-micro/v4@latest

# grpc-gocrawler-gateway插件
go install github.com/grpc-gocrawler-ecosystem/grpc-gocrawler-gateway/v2/protoc-gen-grpc-gocrawler-gateway@latest
go install github.com/grpc-gocrawler-ecosystem/grpc-gocrawler-gateway/v2/protoc-gen-openapiv2@latest

# 下载依赖文件：google/api/annotations.proto
git clone git@github.com:googleapis/googleapis.git
mv googleapis/google  $(go env GOPATH)/src/google

# 将 proto 文件生成协议文件
# 分别是 hello.pb.go、hello.pb.gw.go、hello.pb.micro.go 和 hello_grpc.pb.go。 其中，hello.pb.gw.go 就是 grpc-gocrawler-gateway 插件生成的文件
protoc -I $GOPATH/src  -I .  --micro_out=. --go_out=.  --go-grpc_out=.  --grpc-gocrawler-gateway_out=logtostderr=true,register_func_suffix=Gw:. hello.proto

# Docker启动etcd容器
rm -rf /tmp/etcd-data.tmp && mkdir -p /tmp/etcd-data.tmp && \\
  docker rmi gcr.io/etcd-development/etcd:v3.5.6 || true && \\
  docker run \\
  -p 2379:2379 \\
  -p 2380:2380 \\
  --mount type=bind,source=/tmp/etcd-data.tmp,destination=/etcd-data \\
  --name etcd-gcr-v3.5.6 \\
  gcr.io/etcd-development/etcd:v3.5.6 \\
  /usr/local/bin/etcd \\
  --name s1 \\
  --data-dir /etcd-data \\
  --listen-grpc-client-urls <http://0.0.0.0:2379> \\
  --advertise-grpc-client-urls <http://0.0.0.0:2379> \\
  --listen-peer-urls <http://0.0.0.0:2380> \\
  --initial-advertise-peer-urls <http://0.0.0.0:2380> \\
  --initial-cluster s1=http://0.0.0.0:2380 \\
  --initial-cluster-token tkn \\
  --initial-cluster-state new \\
  --log-level info \\
  --logger zap \\
  --log-outputs stderr
  
# 命令分解
docker run -p 2379:2379 -p 2380:2380 --mount type=bind,source=/tmp/etcd-data.tmp,destination=/etcd-data --name etcd-gcr-v3.5.6 gcr
.io/etcd-development/etcd:v3.5.6

# 静态扫描
go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
golangci-lint run

# 动态扫描
```