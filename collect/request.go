package collect

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// 一个任务实例，
type Task struct {
	Name        string // 用户界面显示的名称（应保证唯一性）
	Url         string
	Cookie      string
	WaitTime    time.Duration
	Reload      bool // 网站是否可以重复爬取
	MaxDepth    int  // 任务最大爬取深度
	Visited     map[string]bool
	VisitedLock sync.Mutex
	Fetcher     Fetcher
	Rule        RuleTree
}

type Context struct {
	Body []byte
	Req  *Request
}

// 单个请求
type Request struct {
	unique   string
	Task     *Task
	Url      string
	Method   string
	Depth    int // 任务的当前深度
	Priority int // 优先级
	RuleName string
}

type ParseResult struct {
	Requesrts []*Request
	Items     []interface{}
}

// 请求的唯一识别码
func (r *Request) Unique() string {
	block := md5.Sum([]byte(r.Url + r.Method))
	return hex.EncodeToString(block[:])
}

func (r *Request) Check() error {
	if r.Depth > r.Task.MaxDepth {
		return errors.New("Max depth limit reached")
	}
	return nil
}
