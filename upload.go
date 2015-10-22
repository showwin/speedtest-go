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

	totalSize := 0
	totalTime := time.Duration(0)
	flg := false
	for i, size := range sizes {
		if flg {
			break
		}
		for j := 0; j < 5; j++ {
			for k := 0; k <= i*5+j; k++ {
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
			totalSize = totalSize + s
			if finish_time.Sub(start_time) > time.Duration(timeout) * time.Second {
				fmt.Println("Timeout")
				flg = true
				break
			}
		}
	}

	return float64(totalSize) * 8 / 1000 / 1000 / totalTime.Seconds()
}
