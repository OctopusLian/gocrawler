package engine

import (
	"go.uber.org/zap"
	"gocrawler/collect"
	"gocrawler/parse/doubangroup"
	"sync"
)

func init() {
	Store.Add(doubangroup.DoubangroupTask)
}

func (c *CrawlerStore) Add(task *collect.Task) {
	c.hash[task.Name] = task
	c.list = append(c.list, task)
}

// 全局蜘蛛种类实例
var Store = &CrawlerStore{
	list: []*collect.Task{},
	hash: map[string]*collect.Task{},
}

type CrawlerStore struct {
	list []*collect.Task
	hash map[string]*collect.Task
}

type Crawler struct {
	out         chan collect.ParseResult
	Visited     map[string]bool // 存储请求访问信息，Visited 中的 Key 是请求的唯一标识，URL + method，并使用 MD5 生成唯一键
	VisitedLock sync.Mutex      // 确保并发安全

	failures    map[string]*collect.Request // 失败请求id -> 失败请求
	failureLock sync.Mutex

	options
}

type Scheduler interface {
	Schedule()
	Push(...*collect.Request)
	Pull() *collect.Request
}

type Schedule struct {
	requestCh   chan *collect.Request // 接收请求
	workerCh    chan *collect.Request // 分配任务给worker
	priReqQueue []*collect.Request    // 优先队列
	reqQueue    []*collect.Request    // 普通队列
	Logger      *zap.Logger
}

func NewEngine(opts ...Option) *Crawler {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	e := &Crawler{}
	e.Visited = make(map[string]bool, 100)
	e.out = make(chan collect.ParseResult)
	e.failures = make(map[string]*collect.Request)
	e.options = options
	return e
}

func NewSchedule() *Schedule {
	s := &Schedule{}
	requestCh := make(chan *collect.Request)
	workerCh := make(chan *collect.Request)
	s.requestCh = requestCh
	s.workerCh = workerCh
	return s
}

func (e *Crawler) Run() {
	go e.Schedule()
	for i := 0; i < e.WorkCount; i++ {
		go e.CreateWork()
	}
	e.HandleResult()
}

// 将请求放入到调度器中
func (s *Schedule) Push(reqs ...*collect.Request) {
	for _, req := range reqs {
		s.requestCh <- req
	}
}

// 从调度器中获取请求
func (s *Schedule) Pull() *collect.Request {
	r := <-s.workerCh
	return r
}

func (s *Schedule) Output() *collect.Request {
	r := <-s.workerCh
	return r
}

func (s *Schedule) Schedule() {
	var req *collect.Request
	var ch chan *collect.Request
	for {
		if req == nil && len(s.priReqQueue) > 0 {
			req = s.priReqQueue[0]
			s.priReqQueue = s.priReqQueue[1:]
			ch = s.workerCh
		}
		if req == nil && len(s.reqQueue) > 0 {
			req = s.reqQueue[0]
			s.reqQueue = s.reqQueue[1:]
			ch = s.workerCh
		}
		select {
		case r := <-s.requestCh:
			if r.Priority > 0 {
				s.priReqQueue = append(s.priReqQueue, r)
			} else {
				s.reqQueue = append(s.reqQueue, r)
			}

		case ch <- req:
			req = nil
			ch = nil
		}
	}
}

// 启动调度器
func (e *Crawler) Schedule() {
	var reqs []*collect.Request
	for _, seed := range e.Seeds {
		task := Store.hash[seed.Name]
		task.Fetcher = seed.Fetcher
		rootreqs := task.Rule.Root()
		for _, req := range rootreqs {
			req.Task = task
		}
		reqs = append(reqs, rootreqs...)
	}
	go e.scheduler.Schedule()
	go e.scheduler.Push(reqs...)
}

func (s *Crawler) CreateWork() {
	for {
		req := s.scheduler.Pull()
		if err := req.Check(); err != nil {
			s.Logger.Error("check failed",
				zap.Error(err),
			)
			continue
		}
		if !req.Task.Reload && s.HasVisited(req) {
			s.Logger.Debug("request has visited",
				zap.String("url:", req.Url),
			)
			continue
		}
		s.StoreVisited(req)

		body, err := req.Task.Fetcher.Get(req)
		if err != nil {
			s.Logger.Error("can't fetch ",
				zap.Error(err),
				zap.String("url", req.Url),
			)
			s.SetFailure(req)
			continue
		}

		if len(body) < 6000 {
			s.Logger.Error("can't fetch ",
				zap.Int("length", len(body)),
				zap.String("url", req.Url),
			)
			s.SetFailure(req)
			continue
		}

		rule := req.Task.Rule.Trunk[req.RuleName]
		result := rule.ParseFunc(&collect.Context{
			body,
			req,
		})

		if len(result.Requesrts) > 0 {
			go s.scheduler.Push(result.Requesrts...)
		}

		s.out <- result
	}
}

func (s *Crawler) HandleResult() {
	for {
		select {
		case result := <-s.out:
			for _, item := range result.Items {
				// todo: store
				s.Logger.Sugar().Info("get result: ", item)
			}
		}
	}
}

func (e *Crawler) HasVisited(r *collect.Request) bool {
	e.VisitedLock.Lock()
	defer e.VisitedLock.Unlock()
	unique := r.Unique()
	return e.Visited[unique]
}

func (e *Crawler) StoreVisited(reqs ...*collect.Request) {
	e.VisitedLock.Lock()
	defer e.VisitedLock.Unlock()

	for _, r := range reqs {
		unique := r.Unique()
		e.Visited[unique] = true
	}
}

func (e *Crawler) SetFailure(req *collect.Request) {
	if !req.Task.Reload {
		e.VisitedLock.Lock()
		unique := req.Unique()
		delete(e.Visited, unique)
		e.VisitedLock.Unlock()
	}
	e.failureLock.Lock()
	defer e.failureLock.Unlock()
	if _, ok := e.failures[req.Unique()]; !ok {
		// 首次失败时，再重新执行一次
		e.failures[req.Unique()] = req
		e.scheduler.Push(req)
	}
	// todo: 失败2次，加载到失败队列中
}
