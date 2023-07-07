package main

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gocrawler/collect"
	"gocrawler/log"
	"gocrawler/proxy"
	"time"
)

func main() {
	plugin, c := log.NewFilePlugin("./log.txt", zapcore.InfoLevel)
	defer c.Close()
	logger := log.NewLogger(plugin)
	logger.Info("log init end")
	proxyURLs := []string{"http://127.0.0.1:7890"}
	p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	if err != nil {
		fmt.Println("RoundRobinProxySwitcher failed")
	}
	url := "https://book.douban.com/subject/1007305/"

	var f collect.Fetcher = collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		Proxy:   p,
	}

	body, err := f.Get(url)
	if err != nil {
		fmt.Println("read content failed:%v", err)
		return
	}
	logger.Info("get content", zap.Int("len", len(body)))
}
