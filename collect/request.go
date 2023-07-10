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
	Url         string
	Cookie      string
	WaitTime    time.Duration
	MaxDepth    int // 任务最大爬取深度
	Visited     map[string]bool
	VisitedLock sync.Mutex
	RootReq     *Request
	Fetcher     Fetcher
}

type Request struct {
	unique    string
	Task      *Task
	Url       string
	Method    string
	Depth     int // 任务的当前深度
	Priority  int // 优先级
	ParseFunc func([]byte, *Request) ParseResult
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
