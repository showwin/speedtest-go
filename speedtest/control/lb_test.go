package control

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSLB(t *testing.T) {
	lb := NewLoadBalancer()
	var a int64 = 0

	lb.Add(func() error {
		atomic.AddInt64(&a, 1)
		time.Sleep(time.Second * 2)
		return errors.New("error")
	}, 2)

	go func() {
		for {
			fmt.Printf("a:%d\n", a)
			time.Sleep(time.Second)
		}
	}()

	wg := sync.WaitGroup{}

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			for {
				lb.Dispatch()
			}
		}()
	}

	wg.Wait()
}

func TestLB(t *testing.T) {
	lb := NewLoadBalancer()
	var a int64 = 0
	var b int64 = 0
	var c int64 = 0
	var d int64 = 0

	lb.Add(func() error {
		atomic.AddInt64(&a, 1)
		time.Sleep(time.Second * 2)
		return nil
	}, 2)

	lb.Add(func() error {
		atomic.AddInt64(&b, 1)
		time.Sleep(time.Second * 2)
		return nil
	}, 1)

	lb.Add(func() error {
		atomic.AddInt64(&c, 1)
		time.Sleep(time.Second * 2)
		fmt.Println("error")
		return errors.New("error")
	}, 1)

	lb.Add(func() error {
		atomic.AddInt64(&d, 1)
		time.Sleep(time.Second * 2)
		return nil
	}, 5)

	wg := sync.WaitGroup{}

	go func() {
		for {
			fmt.Printf("a:%d, b:%d, c:%d, d:%d\n", a, b, c, d)
			time.Sleep(time.Second)
		}
	}()

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			for {
				lb.Dispatch()
			}
		}()
	}

	wg.Wait()
}

func BenchmarkDP(b *testing.B) {
	lb := NewLoadBalancer()
	lb.Add(func() error {
		return nil
	}, 1)
	lb.Add(func() error {
		return nil
	}, 1)
	lb.Add(func() error {
		return nil
	}, 1)
	lb.Add(func() error {
		return nil
	}, 1)
	lb.Add(func() error {
		return nil
	}, 1)

	for i := 0; i < b.N; i++ {
		lb.Dispatch()
	}
}
