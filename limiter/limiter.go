package limiter

import (
	"context"
	"golang.org/x/time/rate"
	"sort"
	"time"
)

type RateLimiter interface {
	Wait(context.Context) error // 环遍历多层限速器 multiLimiter 中所有的限速器并索要令牌，只有当所有的限速器规则都满足后，才会正常执行后续的操作
	Limit() rate.Limit
}

func Per(eventCount int, duration time.Duration) rate.Limit {
	return rate.Every(duration / time.Duration(eventCount))
}

// 聚合多个 RateLimiter，并将速率由小到大排序
func Multi(limiters ...RateLimiter) *MultiLimiter {
	byLimit := func(i, j int) bool {
		return limiters[i].Limit() < limiters[j].Limit()
	}
	sort.Slice(limiters, byLimit)
	return &MultiLimiter{limiters: limiters}
}

type MultiLimiter struct {
	limiters []RateLimiter
}

func (l *MultiLimiter) Wait(ctx context.Context) error {
	for _, l := range l.limiters {
		if err := l.Wait(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (l *MultiLimiter) Limit() rate.Limit {
	return l.limiters[0].Limit()
}
