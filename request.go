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

	//"github.com/cheggaaa/pb"
)

var totalSizeTmp = 0.0
var totalSize = 0.0
var totalTime = time.Duration(0)
var finished = false

func initialize() {
	totalSizeTmp = 0.0
	totalSize = 0.0
	totalTime = time.Duration(0)
	finished = false
}

func SpeedTest(kind string, url string) float64 {
	initialize()
	wg := new(sync.WaitGroup)
	m := new(sync.Mutex)
	startTime := time.Now()
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go loopRequest(wg, m, startTime, kind, url)
	}
	wg.Wait()
	fmt.Println("total size")
	fmt.Println(totalSize)
	return totalSize * 8 / totalTime.Seconds()
}

func loopRequest(wg *sync.WaitGroup, m *sync.Mutex, startTime time.Time, kind string, url string) {
	if kind == "download" {
		for {
			downloadRequest(wg, m, startTime, url)
			if finished {
				break
			}
		}
	} else {
		for {
			uploadRequest(wg, m, startTime, url)
			if finished {
				break
			}
		}
	}
}

func downloadRequest(wg *sync.WaitGroup, m *sync.Mutex, startTime time.Time, dlUrl string) {
	// sizes := [...]int{350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
	size := 1000
	url := dlUrl + "/random" + strconv.Itoa(size) + "x" + strconv.Itoa(size) + ".jpg"
	resp, err := http.Get(url)
	CheckError(err)
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)

	dSize := 2 * size * size
	updateTotalSize(dSize, wg, m, startTime)
}

func uploadRequest(wg *sync.WaitGroup, m *sync.Mutex, startTime time.Time, ulUrl string) {
	// sizes := [...]int{100, 300, 500, 800, 1000, 2000, 3000, 4000} //kB
	size := 1000
	v := url.Values{}
	v.Add("content", strings.Repeat("0", size*1000-160))

	resp, err := http.PostForm(ulUrl, v)
	CheckError(err)
	defer resp.Body.Close()
	r_body, _ := ioutil.ReadAll(resp.Body)

	uSize, _ := strconv.Atoi(string(r_body)[5:])
	updateTotalSize(uSize, wg, m, startTime)
}

func updateTotalSize(size int, wg *sync.WaitGroup, m *sync.Mutex, startTime time.Time) {
	m.Lock()
	defer m.Unlock()
	totalSizeTmp = totalSizeTmp + float64(size)/1000/1000 //MB
	if time.Now().After(startTime.Add(10 * time.Second)) {
		wg.Done()
		if finished == false {
			finished = true
			totalSize = totalSizeTmp
			totalTime = time.Now().Sub(startTime)
		}
	}
}
