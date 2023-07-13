package main

import (
	"context"
	"fmt"
	pb "github.com/dreamerjackson/crawler/proto/greeter"
	etcdReg "github.com/go-micro/plugins/v4/registry/etcd"
	gs "github.com/go-micro/plugins/v4/server/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gocrawler/collect"
	"gocrawler/engine"
	"gocrawler/limiter"
	"gocrawler/log"
	"gocrawler/proxy"
	"gocrawler/storage"
	"gocrawler/storage/sqlstorage"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"net/http"
	"time"
)

func main() {
	plugin := log.NewStdoutPlugin(zapcore.DebugLevel)
	logger := log.NewLogger(plugin)
	logger.Info("log init end")

	// set zap global logger
	zap.ReplaceGlobals(logger)

	// proxy
	proxyURLs := []string{"http://127.0.0.1:7890"}
	var p proxy.ProxyFunc
	var err error
	p, err = proxy.RoundRobinProxySwitcher(proxyURLs...)
	if err != nil {
		logger.Error("RoundRobinProxySwitcher failed")
		return
	}

	// fetcher
	var f collect.Fetcher = &collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		Logger:  logger,
		Proxy:   p,
	}

	// storage
	var storage storage.Storage
	storage, err = sqlstorage.New(
		sqlstorage.WithSqlUrl("root:mysql123@tcp(localhost:3306)/gocrawler?charset=utf8"),
		sqlstorage.WithLogger(logger.Named("sqlDB")),
		sqlstorage.WithBatchCount(2),
	)
	if err != nil {
		logger.Error("create sqlstorage failed")
		return
	}

	// speed limiter
	secondLimit := rate.NewLimiter(limiter.Per(1, 2*time.Second), 1)   //2秒钟1个
	minuteLimit := rate.NewLimiter(limiter.Per(20, 1*time.Minute), 20) //60秒20个
	multiLimiter := limiter.MultiLimiter(secondLimit, minuteLimit)

	// init tasks
	var seeds = make([]*collect.Task, 0, 1000)
	seeds = append(seeds, &collect.Task{
		Property: collect.Property{
			Name: "douban_book_list",
		},
		Fetcher: f,
		Storage: storage,
		Limit:   multiLimiter,
	})

	s := engine.NewEngine(
		engine.WithFetcher(f),
		engine.WithLogger(logger),
		engine.WithWorkCount(5),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
	)
	// worker start
	s.Run()

	// start http proxy to GRPC
	go HandleHTTP()

	// start grpc server
	reg := etcdReg.NewRegistry(
		registry.Addrs(":2379"),
	)
	service := micro.NewService(
		micro.Server(gs.NewServer(
			server.Id("1"),
		)),
		micro.Address(":9090"),
		micro.Registry(reg),
		micro.Name("go.micro.server.worker"),
	)
	service.Init()
	pb.RegisterGreeterHandler(service.Server(), new(Greeter))
	if err := service.Run(); err != nil {
		logger.Fatal("grpc server stop")
	}
}

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	rsp.Greeting = "Hello " + req.Name
	return nil
}

func HandleHTTP() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	err := pb.RegisterGreeterGwFromEndpoint(ctx, mux, "localhost:9090", opts)
	if err != nil {
		fmt.Println(err)
	}

	http.ListenAndServe(":8080", mux)
}
