package collect

import (
	"errors"
	"time"
)

type Request struct {
	Url       string
	Cookie    string
	WaitTime  time.Duration
	Depth     int // 任务的当前深度
	MaxDepth  int // 任务最大爬取深度
	ParseFunc func([]byte, *Request) ParseResult
}

type ParseResult struct {
	Requesrts []*Request
	Items     []interface{}
}

func (r *Request) Check() error {
	if r.Depth > r.MaxDepth {
		return errors.New("Max depth limit reached")
	}
	return nil
}
