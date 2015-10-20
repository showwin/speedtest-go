package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/cheggaaa/pb"
)

func DownloadSpeed(dlUrl string) float64 {
	fmt.Println("Testing Download Speed ...")
	count := 40 * (40 + 1) / 2
	bar := pb.StartNew(count)
	bar.ShowBar = false
	bar.ShowCounters = false
	sizes := [...]int{350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
	urls := [40]string{}
	for i, size := range sizes {
		for j := 0; j < 4; j++ {
			urls[i*4+j] = dlUrl + "/random" + strconv.Itoa(size) + "x" + strconv.Itoa(size) + ".jpg"
		}
	}

	totalTime := time.Duration(0)
	for i, url := range urls {
		for j := 0; j <= i; j++ {
			bar.Increment()
		}
		start_time := time.Now()
		resp, err := http.Get(url)
		CheckError(err)
		ioutil.ReadAll(resp.Body)
		finish_time := time.Now()
		defer resp.Body.Close()

		totalTime = totalTime + finish_time.Sub(start_time)
	}

	sumSize := 0.0
	for _, size := range sizes {
		sumSize = sumSize + 4*2*float64(size)*float64(size)/1000/1000
	}

	return sumSize * 8 / totalTime.Seconds()
}
