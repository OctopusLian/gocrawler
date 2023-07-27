package main

import (
	"gocrawler/cmd"
	_ "net/http/pprof"
)

func main() {
	cmd.Execute()
}
