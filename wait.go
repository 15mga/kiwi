package kiwi

import (
	"github.com/15mga/kiwi/util"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func WaitGroup(secs int, wg *sync.WaitGroup, ticker util.FnInt) bool {
	tc := time.NewTicker(time.Second)
	ch := make(chan struct{})
	defer func() {
		tc.Stop()
	}()
	go func() {
		wg.Wait()
		close(ch)
	}()
	for {
		select {
		case <-ch:
			return true
		case <-tc.C:
			secs--
			if secs == 0 {
				return false
			}
			if ticker != nil {
				ticker(secs)
			}
		}
	}
}

func GoWaitGroup(secs int, wg *sync.WaitGroup, over util.FnBool, ticker util.FnInt) {
	go over(WaitGroup(secs, wg, ticker))
}

type waitInfo struct {
	name string
	fn   util.Fn
}

var (
	_WaitExitInfos = make([]*waitInfo, 0, 1)
)

func BeforeExitFn(name string, fn util.Fn) {
	_WaitExitInfos = append(_WaitExitInfos, &waitInfo{
		name: name,
		fn:   fn,
	})
}

func BeforeExitCh(name string) chan<- struct{} {
	ch := make(chan struct{})
	BeforeExitFn(name, func() {
		<-ch
	})
	return ch
}

func WaitExit() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case <-util.Ctx().Done():
		Info("context done", nil)
	case s := <-signalCh:
		Info("signal notify", util.M{
			"signal": s,
		})
		util.Cancel()
	}

	waitCh := make(chan struct{})
	go func() {
		count := len(_WaitExitInfos)
		if count == 0 {
			close(waitCh)
			return
		}
		nameCh := make(chan string, count)
		status := make(util.M, count)
		for _, info := range _WaitExitInfos {
			status[info.name] = false
			wInfo := info
			go func() {
				Info("wait exit", util.M{
					"name": wInfo.name,
				})
				wInfo.fn()
				nameCh <- wInfo.name
			}()
		}
		ticker := time.NewTicker(time.Second)
		tickCount := 0
		for {
			select {
			case <-ticker.C:
				Info("exit status", util.M{
					"status": status,
					"count":  tickCount,
				})
				tickCount++
			case name := <-nameCh:
				Info("exit", util.M{
					"name": name,
				})
				status[name] = true
				count--
				if count == 0 {
					ticker.Stop()
					close(waitCh)
					return
				}
			}
		}
	}()

	timeout := time.NewTimer(time.Second * 60)
	select {
	case <-timeout.C:
		Info("exit timeout", nil)
	case <-waitCh:
		timeout.Stop()
		Info("exit complete", nil)
	}
}
