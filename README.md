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
# grpc
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# go-micro
go install github.com/asim/go-micro/cmd/protoc-gen-micro/v4@latest

# grpc-gateway插件
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

# 下载依赖文件：google/api/annotations.proto
git clone git@github.com:googleapis/googleapis.git
mv googleapis/google  $(go env GOPATH)/src/google

# 将 proto 文件生成协议文件
# 分别是 hello.pb.go、hello.pb.gw.go、hello.pb.micro.go 和 hello_grpc.pb.go。 其中，hello.pb.gw.go 就是 grpc-gateway 插件生成的文件
protoc -I $GOPATH/src  -I .  --micro_out=. --go_out=.  --go-grpc_out=.  --grpc-gateway_out=logtostderr=true,register_func_suffix=Gw:. hello.proto

```