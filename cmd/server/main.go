package main

import "github.com/yourname/github-sentinel/internal/scheduler"

func main() {
s := &scheduler.Scheduler{}
s.Start()
select {}
}
