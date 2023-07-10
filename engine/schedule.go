package engine

import (
	"go.uber.org/zap"
	"gocrawler/collect"
)

type Schedule struct {
	requestCh chan *collect.Request    // 接收请求
	workerCh  chan *collect.Request    // 分配任务给worker
	out       chan collect.ParseResult // 处理爬取后的数据，完成下一步存储的操作
	options
}

type Config struct {
	WorkCount int
	Fetcher   collect.Fetcher
	Logger    *zap.Logger
	Seeds     []*collect.Request
}

func NewSchedule(opts ...Option) *Schedule {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	s := &Schedule{}
	s.options = options
	return s
}

func (s *Schedule) Run() {
	requestCh := make(chan *collect.Request)
	workerCh := make(chan *collect.Request)
	out := make(chan collect.ParseResult)
	s.requestCh = requestCh
	s.workerCh = workerCh
	s.out = out
	go s.Schedule()
	for i := 0; i < s.WorkCount; i++ {
		go s.CreateWork()
	}
	s.HandleResult()
}

func (s *Schedule) Schedule() {
	var reqQueue = s.Seeds
	go func() {
		for { // 让调度器循环往复地获取外界的爬虫任务，并将任务分发到 worker 中
			var req *collect.Request
			var ch chan *collect.Request

			if len(reqQueue) > 0 { // 如果任务队列 reqQueue 大于 0，意味着有爬虫任务
				req = reqQueue[0]       // 获取队列中第一个任务
				reqQueue = reqQueue[1:] // 并将其剔除出队列
				ch = s.workerCh
			}
			select {
			case r := <-s.requestCh: // 接收来自外界的请求
				reqQueue = append(reqQueue, r) // 将请求存储到 reqQueue 队列中

			case ch <- req: // 将任务发送到 workerCh 通道中，等待 worker 接收
			}
		}
	}()
}

func (s *Schedule) CreateWork() {
	for {
		r := <-s.workerCh             // 接收到调度器分配的任务
		body, err := s.Fetcher.Get(r) // 访问服务器
		if err != nil {
			s.Logger.Error("can't fetch ",
				zap.Error(err),
			)
			continue
		}
		result := r.ParseFunc(body, r) // 解析服务器返回的数据
		s.out <- result                // 将返回的数据发送到 out 通道中，方便后续的处理
	}
}

func (s *Schedule) HandleResult() {
	for {
		select {
		case result := <-s.out: // 接收所有 worker 解析后的数据
			for _, req := range result.Requesrts {
				s.requestCh <- req // 要进一步爬取的 Requests 列表将全部发送回 s.requestCh 通道
			}
			for _, item := range result.Items {
				// todo: store
				s.Logger.Sugar().Info("get result", item)
			}
		}
	}
}
