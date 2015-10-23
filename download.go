package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
	"sync"

	//"github.com/cheggaaa/pb"
)

var totalSizeTmp = 0.0
var totalSize = 0.0
var totalTime = time.Duration(0)
var finished = false

func DownloadSpeed(dlUrl string) float64 {
	wg := new(sync.WaitGroup)
	m := new(sync.Mutex)
	startTime := time.Now()
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go loopRequest(wg, m, startTime, dlUrl)
	}
	wg.Wait()
	fmt.Println("total size")
	fmt.Println(totalSize)
	return totalSize * 8 / totalTime.Seconds()
}

func loopRequest(wg *sync.WaitGroup, m *sync.Mutex, startTime time.Time, dlUrl string) {
	for {
		sendRequest(wg, m, startTime, dlUrl)
		if finished {
			break
		}
	}
}

func sendRequest(wg *sync.WaitGroup, m *sync.Mutex, startTime time.Time, dlUrl string) {
	size := 1000
	url := dlUrl + "/random" + strconv.Itoa(size) + "x" + strconv.Itoa(size) + ".jpg"
	resp, err := http.Get(url)
	CheckError(err)
	ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	updateTotalSize(size, wg, m, startTime)
}

func updateTotalSize(size int, wg *sync.WaitGroup, m *sync.Mutex, startTime time.Time) {
	m.Lock()
	defer m.Unlock()
	totalSizeTmp = totalSizeTmp + 2*float64(size)*float64(size)/1000/1000
	if time.Now().After(startTime.Add(10 * time.Second)) {
		wg.Done()
		if finished == false {
			finished = true
			totalSize = totalSizeTmp
			totalTime = time.Now().Sub(startTime)
		}
	}
}

/*

func aaa() {
	fmt.Println("Testing Download Speed ...")
	count := 40 * (40 + 1) / 2
	bar := pb.StartNew(count)
	bar.ShowBar = false
	bar.ShowCounters = false
	sizes := [...]int{350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}

	totalTime := time.Duration(0)
	totalSize := 0.0
	flg := false
	for i, size := range sizes {
		if flg {
			break
		}
		for j := 0; j < 4; j++ {
			for k := 0; k <= i*4+j; k++ {
				bar.Increment()
			}
			url := dlUrl + "/random" + strconv.Itoa(size) + "x" + strconv.Itoa(size) + ".jpg"
			start_time := time.Now()
			resp, err := http.Get(url)
			CheckError(err)
			ioutil.ReadAll(resp.Body)
			finish_time := time.Now()
			defer resp.Body.Close()

			totalTime = totalTime + finish_time.Sub(start_time)
			totalSize = totalSize + 2*float64(size)*float64(size)/1000/1000
			if finish_time.Sub(start_time) > time.Duration(timeout) * time.Second {
				fmt.Println("Timeout")
				flg = true
				break
			}
		}
	}

	return totalSize * 8 / totalTime.Seconds()
}

*/
