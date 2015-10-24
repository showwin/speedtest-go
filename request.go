package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var totalSizeTmp = 0.0
var totalSize = 0.0
var latency = time.Duration(0)
var extraTime = time.Duration(0)
var startTime = time.Now()
var on = time.Time{}
var off = time.Time{}
var exitTime = time.Time{}
var finished = false
var workload = 0
var dlSizes = [...]int{350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
var ulSizes = [...]int{100, 300, 500, 800, 1000, 1500, 2500, 3000, 3500, 4000} //kB
var client = http.Client{}

func initialize(kind string, sUrl string) {
	totalSizeTmp = 0.0
	totalSize = 0.0
	latency = pingTest(sUrl)
	extraTime = latency*20
	finished = false
	workload = 0
	startTime = time.Now()
	on = startTime.Add(6 * time.Second)
	off = startTime.Add(18 * time.Second).Add(extraTime)
	exitTime = startTime.Add(22 * time.Second).Add(extraTime)
	fmt.Printf(kind+" Test: ")
}

func SpeedTest(kind string, sUrl string) float64 {
	initialize(kind, sUrl)
	wg := new(sync.WaitGroup)
	m := new(sync.Mutex)

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go loopRequest(wg, m, kind, sUrl)
	}
	wg.Wait()
	fmt.Printf("\n")

	return totalSize * 8 / (12 + extraTime.Seconds())
}

func loopRequest(wg *sync.WaitGroup, m *sync.Mutex, kind string, sUrl string) {
	if kind == "Download" {
		dlUrl := strings.Split(sUrl, "/upload")[0]
		for {
			downloadRequest(wg, m, dlUrl)
			if finished {
				break
			}
		}
	} else {
		for {
			uploadRequest(wg, m, sUrl)
			if finished {
				break
			}
		}
	}
}

func downloadRequest(wg *sync.WaitGroup, m *sync.Mutex, dlUrl string) {
	size := dlSizes[workload/3]
	url := dlUrl + "/random" + strconv.Itoa(size) + "x" + strconv.Itoa(size) + ".jpg"

	sTime := time.Now()
	resp, _ := client.Get(url)
	ioutil.ReadAll(resp.Body)
	fTime := time.Now()
	defer resp.Body.Close()

	updateTotalSize(2*size*size, wg, m, fTime.Sub(sTime))
}

func uploadRequest(wg *sync.WaitGroup, m *sync.Mutex, ulUrl string) {
	size := ulSizes[workload/3]
	v := url.Values{}
	v.Add("content", strings.Repeat("0", size*1000-160))

	sTime := time.Now()
	resp, _ := client.PostForm(ulUrl, v)
	defer resp.Body.Close()
	rBody, _ := ioutil.ReadAll(resp.Body)
	fTime := time.Now()

	uSize, _ := strconv.Atoi(string(rBody)[5:])
	updateTotalSize(uSize, wg, m, fTime.Sub(sTime))
}

func updateTotalSize(size int, wg *sync.WaitGroup, m *sync.Mutex, execTime time.Duration) {
	m.Lock()
	defer m.Unlock()
	fmt.Printf(".")
	if time.Now().After(on) && time.Now().Before(off) {
		totalSizeTmp = totalSizeTmp + float64(size)/1000/1000 //MB
	}
	if execTime < (2000 * time.Millisecond) + latency*4 {
		incrementWorkload()
	} else if execTime > (10000 * time.Millisecond) + latency*4 {
		decrementWorkload()
	}
	if time.Now().After(exitTime) {
		wg.Done()
		if finished == false {
			finished = true
			totalSize = totalSizeTmp
		}
	}
}

func incrementWorkload() {
	workload++
	if workload > 9*3 {
		workload = 9*3
	}
}

func decrementWorkload() {
	workload--
}

func pingTest(sUrl string) time.Duration {
	pingUrl := strings.Split(sUrl, "/upload")[0] + "/latency.txt"

	l := time.Duration(0)
	for i := 0; i < 3; i++ {
		sTime := time.Now()
		resp, err := http.Get(pingUrl)
		fTime := time.Now()
		CheckError(err)
		defer resp.Body.Close()
		l = l + fTime.Sub(sTime)
	}

	fmt.Println("latency:", (l / 6.0))
	return l / 6.0
}
