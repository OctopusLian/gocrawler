package engine

import (
	"github.com/robertkrimen/otto"
	"go.uber.org/zap"
	"gocrawler/parse/doubanbook"
	"gocrawler/parse/doubangroup"
	"gocrawler/parse/doubangroupjs"
	"gocrawler/spider"
	"runtime/debug"
	"sync"
)

func init() {
	Store.Add(doubangroup.DoubangroupTask)
	Store.Add(doubanbook.DoubanBookTask)
	Store.AddJSTask(doubangroupjs.DoubangroupJSTask)
}

func (c *CrawlerStore) Add(task *spider.Task) {
	c.Hash[task.Name] = task
	c.list = append(c.list, task)
}

type mystruct struct {
	Name string
	Age  int
}

// 用于动态规则添加请求。
func AddJsReqs(jreqs []map[string]interface{}) []*spider.Request {
	reqs := make([]*spider.Request, 0)

	for _, jreq := range jreqs {
		req := &spider.Request{}
		u, ok := jreq["Url"].(string)
		if !ok {
			return nil
		}
		req.URL = u
		req.RuleName, _ = jreq["RuleName"].(string)
		req.Method, _ = jreq["Method"].(string)
		req.Priority, _ = jreq["Priority"].(int64)
		reqs = append(reqs, req)
	}
	return reqs
}

// 用于动态规则添加请求。
func AddJsReq(jreq map[string]interface{}) []*spider.Request {
	reqs := make([]*spider.Request, 0)
	req := &spider.Request{}
	u, ok := jreq["Url"].(string)
	if !ok {
		return nil
	}
	req.URL = u
	req.RuleName, _ = jreq["RuleName"].(string)
	req.Method, _ = jreq["Method"].(string)
	req.Priority, _ = jreq["Priority"].(int64)
	reqs = append(reqs, req)
	return reqs
}

func (c *CrawlerStore) AddJSTask(m *spider.TaskModle) {
	task := &spider.Task{
		//Property: m.Property,
	}

	task.Rule.Root = func() ([]*spider.Request, error) {
		vm := otto.New()
		vm.Set("AddJsReq", AddJsReqs)
		v, err := vm.Eval(m.Root)
		if err != nil {
			return nil, err
		}
		e, err := v.Export()
		if err != nil {
			return nil, err
		}
		return e.([]*spider.Request), nil
	}

	for _, r := range m.Rules {
		paesrFunc := func(parse string) func(ctx *spider.Context) (spider.ParseResult, error) {
			return func(ctx *spider.Context) (spider.ParseResult, error) {
				vm := otto.New()
				vm.Set("ctx", ctx)
				v, err := vm.Eval(parse)
				if err != nil {
					return spider.ParseResult{}, err
				}
				e, err := v.Export()
				if err != nil {
					return spider.ParseResult{}, err
				}
				if e == nil {
					return spider.ParseResult{}, err
				}
				return e.(spider.ParseResult), err
			}
		}(r.ParseFunc)
		if task.Rule.Trunk == nil {
			task.Rule.Trunk = make(map[string]*spider.Rule, 0)
		}
		task.Rule.Trunk[r.Name] = &spider.Rule{
			ParseFunc: paesrFunc,
		}
	}

	c.Hash[task.Name] = task
	c.list = append(c.list, task)
}

// 全局爬虫任务实例
var Store = &CrawlerStore{
	list: []*spider.Task{},
	Hash: map[string]*spider.Task{},
}

func GetFields(taskName string, ruleName string) []string {
	return Store.Hash[taskName].Rule.Trunk[ruleName].ItemFields
}

type CrawlerStore struct {
	list []*spider.Task
	Hash map[string]*spider.Task
}

type Crawler struct {
	out         chan spider.ParseResult
	Visited     map[string]bool // 存储请求访问信息，Visited 中的 Key 是请求的唯一标识，URL + method，并使用 MD5 生成唯一键
	VisitedLock sync.Mutex      // 确保并发安全

	failures    map[string]*spider.Request // 失败请求id -> 失败请求
	failureLock sync.Mutex

	options
}

type Scheduler interface {
	Schedule()
	Push(...*spider.Request)
	Pull() *spider.Request
}

type Schedule struct {
	requestCh   chan *spider.Request // 接收请求
	workerCh    chan *spider.Request // 分配任务给worker
	priReqQueue []*spider.Request    // 优先队列
	reqQueue    []*spider.Request    // 普通队列
	Logger      *zap.Logger
}

func NewEngine(opts ...Option) *Crawler {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	e := &Crawler{}
	e.Visited = make(map[string]bool, 100)
	e.out = make(chan spider.ParseResult)
	e.failures = make(map[string]*spider.Request)
	e.options = options
	return e
}

func NewSchedule() *Schedule {
	s := &Schedule{}
	requestCh := make(chan *spider.Request)
	workerCh := make(chan *spider.Request)
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
func (s *Schedule) Push(reqs ...*spider.Request) {
	for _, req := range reqs {
		s.requestCh <- req
	}
}

// 从调度器中获取请求
func (s *Schedule) Pull() *spider.Request {
	r := <-s.workerCh
	return r
}

func (s *Schedule) Output() *spider.Request {
	r := <-s.workerCh
	return r
}

func (s *Schedule) Schedule() {
	var req *spider.Request
	var ch chan *spider.Request
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

		// 请求校验
		if req != nil {
			if err := req.Check(); err != nil {
				zap.L().Debug("check failed",
					zap.Error(err),
				)
				req = nil
				ch = nil
				continue
			}
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
func (c *Crawler) Schedule() {
	var reqs []*spider.Request

	for _, task := range c.Seeds {
		t, ok := Store.Hash[task.Name]
		if !ok {
			c.Logger.Error("can not find preset tasks", zap.String("task name", task.Name))
			continue
		}
		task.Rule = t.Rule
		rootreqs, err := task.Rule.Root()
		if err != nil {
			c.Logger.Error("get root failed",
				zap.Error(err),
			)

			continue
		}
		for _, req := range rootreqs {
			req.Task = task
		}
		reqs = append(reqs, rootreqs...)
	}
	go c.scheduler.Schedule()
	go c.scheduler.Push(reqs...)
}

func (s *Crawler) CreateWork() {
	defer func() {
		if err := recover(); err != nil {
			s.Logger.Error("worker panic",
				zap.Any("err", err),
				zap.String("stack", string(debug.Stack())))
		}
	}()
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
				zap.String("url:", req.URL),
			)
			continue
		}
		s.StoreVisited(req)

		body, err := req.Fetch()
		if err != nil {
			s.Logger.Error("can't fetch ",
				zap.Error(err),
				zap.String("url", req.URL),
			)
			s.SetFailure(req)
			continue
		}

		if len(body) < 6000 {
			s.Logger.Error("can't fetch ",
				zap.Int("length", len(body)),
				zap.String("url", req.URL),
			)
			s.SetFailure(req)
			continue
		}

		rule := req.Task.Rule.Trunk[req.RuleName]
		result, err := rule.ParseFunc(&spider.Context{
			body,
			req,
		})
		if err != nil {
			s.Logger.Error("ParseFunc failed ",
				zap.Error(err),
				zap.String("url", req.URL),
			)
			continue
		}

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
				switch d := item.(type) {
				case *spider.DataCell:
					name := d.GetTaskName()
					task := Store.Hash[name]
					task.Storage.Save(d)
				}
				s.Logger.Sugar().Info("get result: ", item)
			}
		}
	}
}

func (e *Crawler) HasVisited(r *spider.Request) bool {
	e.VisitedLock.Lock()
	defer e.VisitedLock.Unlock()
	unique := r.Unique()
	return e.Visited[unique]
}

func (e *Crawler) StoreVisited(reqs ...*spider.Request) {
	e.VisitedLock.Lock()
	defer e.VisitedLock.Unlock()

	for _, r := range reqs {
		unique := r.Unique()
		e.Visited[unique] = true
	}
}

func (e *Crawler) SetFailure(req *spider.Request) {
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
