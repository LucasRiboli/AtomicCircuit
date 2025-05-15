package main

import (
	"net/http"
	"sync"
	"sync/atomic"
)

const (
	open     = "Open"
	close    = "Close"
	halfOpen = "HalfOpen"
)

var state = "Close"
var QntError atomic.Uint64
var QntReq atomic.Uint64

var links = []string{
	"https://example.com/resource/1",
	"https://example.com/resource/2",
	"https://example.com/resource/3",
	"https://example.com/resource/4",
	"https://example.com/resource/5",
	"https://invalid-domain-10.fake/error",
	"https://invalid-domain-11.fake/error",
	"https://invalid-domain-12.fake/error",
	"https://invalid-domain-13.fake/error",
	"https://invalid-domain-14.fake/error",
	"https://invalid-domain-15.fake/error",
	"https://invalid-domain-16.fake/error",
	"https://invalid-domain-17.fake/error",
	"https://invalid-domain-18.fake/error",
	"https://invalid-domain-19.fake/error",
	"https://invalid-domain-20.fake/error",
	"https://example.com/resource/1",
	"https://example.com/resource/2",
	"https://example.com/resource/3",
	"https://example.com/resource/4",
	"https://example.com/resource/5",
	"https://example.com/resource/5",
	"https://example.com/resource/1",
	"https://example.com/resource/2",
	"https://example.com/resource/3",
	"https://example.com/resource/4",
	"https://example.com/resource/5",
	"https://example.com/resource/5",
}

func main() {
	var wg sync.WaitGroup

	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func() {
			for _, link := range links {
				stateMachineHandler(link)
				println("---")
				println(state)
				println(QntReq.Load())
				println(QntError.Load())
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func stateMachineHandler(url string) string {
	div5 := QntReq.Load() % 3
	if div5 == 0 && QntError.Load() > 5 {
		state = "HalfOpen"
	} else if QntError.Load() > 5 {
		state = "Open"
	} else {
		state = "Close"
	}

	switch state {
	case open:
		QntReq.Add(1)
		return "Open State wait service estabilize"
	case close:
		err := MiddlayercallGrpc(url)
		if err != nil {
			QntError.Add(1)
			QntReq.Add(1)
			return err.Error()
		}
		QntReq.Add(1)
		return "Ok"
	case halfOpen:
		err := MiddlayercallGrpc(url)
		if err != nil {
			QntError.Add(1)
			QntReq.Add(1)
			return err.Error()
		}
		AtomicDecrementUint64(&QntError)
		QntReq.Add(1)
	}
	return ""
}

func AtomicDecrementUint64(v *atomic.Uint64) {
	for {
		old := v.Load()
		if old == 0 {
			return // or panic/log/etc
		}
		if v.CompareAndSwap(old, old-1) {
			return
		}
	}
}

func MiddlayercallGrpc(url string) error {
	// after implements a interface now only get fuck it
	_, err := http.Get(url)
	if err != nil {
		return err
	}
	return nil
}
