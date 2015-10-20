package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
)

func UploadSpeed(ulUrl string) float64 {
	fmt.Println("Testing Upload Speed ...")
	count := 40 * (40 + 1) / 2
	bar := pb.StartNew(count)
	bar.ShowBar = false
	bar.ShowCounters = false
	sizes := [...]int{100, 300, 500, 800, 1000, 2000, 3000, 4000} //kB

	testSizes := [40]int{}
	for i, size := range sizes {
		for j := 0; j < 5; j++ {
			testSizes[i*5+j] = size
		}
	}

	sumSize := 0
	totalTime := time.Duration(0)
	for i, size := range testSizes {
		for j := 0; j <= i; j++ {
			bar.Increment()
		}
		v := url.Values{}
		v.Add("content", strings.Repeat("0", size*1000-160))

		start_time := time.Now()
		resp, err := http.PostForm(ulUrl, v)
		CheckError(err)
		r_body, _ := ioutil.ReadAll(resp.Body)
		finish_time := time.Now()
		defer resp.Body.Close()

		totalTime = totalTime + finish_time.Sub(start_time)
		s, _ := strconv.Atoi(string(r_body)[5:])
		sumSize = sumSize + s
	}

	return float64(sumSize) * 8 / 1000 / 1000 / totalTime.Seconds()
}
