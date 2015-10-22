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
