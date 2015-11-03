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

var dlSizes = [...]int{350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
var ulSizes = [...]int{100, 300, 500, 800, 1000, 1500, 2500, 3000, 3500, 4000} //kB
var client = http.Client{}

func DownloadTest(sUrl string) float64 {
	dlUrl := strings.Split(sUrl, "/upload")[0]
	latency := pingTest(sUrl)
	fmt.Printf("Download WarmUp: ")
	wg := new(sync.WaitGroup)

	// worm up
	sTime := time.Now()
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go dlWarmUp(wg, dlUrl)
	}
	wg.Wait()
	fTime := time.Now()
	// 1.125MB for each request (750 * 750 * 2)
	speed := 1.125 * 8 * 2 / fTime.Sub(sTime.Add(latency)).Seconds()
	fmt.Printf("%5.2f Mbit/s\n", speed)

	// decide workload by warm up speed
	workload := 0
	if 10.0 < speed {
		workload = 16
	} else if 1.0 < speed {
		workload = 8
	} else {
		workload = 4
	}

	// speedtest
	fmt.Printf("Download Test: ")
	sTime = time.Now()
	for i := 0; i < workload; i++ {
		wg.Add(1)
		go downloadRequest(wg, dlUrl)
	}
	wg.Wait()
	fTime = time.Now()
	fmt.Printf("\n")

	// 4.5MB for each request (width(1500) * height(1500) * 2)
	return 4.5 * 8 * float64(workload) / fTime.Sub(sTime).Seconds()
}

func UploadTest(sUrl string) float64 {
	latency := pingTest(sUrl)
	fmt.Printf("Upload WarmUp: ")
	wg := new(sync.WaitGroup)

	// worm up
	sTime := time.Now()
	wg = new(sync.WaitGroup)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go ulWarmUp(wg, sUrl)
	}
	wg.Wait()
	fTime := time.Now()
	// 1.0 MB for each request
	speed := 1.0 * 8 * 2 / fTime.Sub(sTime.Add(latency)).Seconds()
	fmt.Printf("%5.2f Mbit/s\n", speed)

	// decide workload by warm up speed
	workload := 0
	if 10.0 < speed {
		workload = 16
	} else if 1.0 < speed {
		workload = 8
	} else {
		workload = 4
	}

	// speedtest
	fmt.Printf("Upload Test: ")
	sTime = time.Now()
	for i := 0; i < workload; i++ {
		wg.Add(1)
		go uploadRequest(wg, sUrl)
	}
	wg.Wait()
	fTime = time.Now()
	fmt.Printf("\n")

	// 4.0 MB for each request
	return 4 * 8 * float64(workload) / fTime.Sub(sTime).Seconds()
}

func dlWarmUp(wg *sync.WaitGroup, dlUrl string) {
	size := dlSizes[2]
	url := dlUrl + "/random" + strconv.Itoa(size) + "x" + strconv.Itoa(size) + ".jpg"

	resp, _ := client.Get(url)
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)

	wg.Done()
}

func ulWarmUp(wg *sync.WaitGroup, ulUrl string) {
	size := ulSizes[4]
	v := url.Values{}
	v.Add("content", strings.Repeat("0123456789", size*100-51))

	resp, _ := client.PostForm(ulUrl, v)
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)

	wg.Done()
}

func downloadRequest(wg *sync.WaitGroup, dlUrl string) {
	size := dlSizes[4]
	url := dlUrl + "/random" + strconv.Itoa(size) + "x" + strconv.Itoa(size) + ".jpg"

	resp, _ := client.Get(url)
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)

	fmt.Printf(".")
	wg.Done()
}

func uploadRequest(wg *sync.WaitGroup, ulUrl string) {
	size := ulSizes[9]
	v := url.Values{}
	v.Add("content", strings.Repeat("0123456789", size*100-51))

	resp, _ := client.PostForm(ulUrl, v)
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)

	fmt.Printf(".")
	wg.Done()
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
