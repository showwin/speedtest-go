package speedtest

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkDataManager_NewDataChunk(b *testing.B) {
	dmp := NewDataManager()
	dmp.DataGroup = make([]*DataChunk, 64)
	for i := 0; i < b.N; i++ {
		dmp.NewDataChunk()

	}
}

func TestDynamicRate(t *testing.T) {

	oldDownTotal := GlobalDataManager.GetTotalDownload()
	oldUpTotal := GlobalDataManager.GetTotalUpload()

	go func() {
		for i := 0; i < 20; i++ {
			time.Sleep(time.Second)
			newDownTotal := GlobalDataManager.GetTotalDownload()
			newUpTotal := GlobalDataManager.GetTotalUpload()

			downRate := float64(newDownTotal-oldDownTotal) * 8 / 1000 / 1000
			upRate := float64(newUpTotal-oldUpTotal) * 8 / 1000 / 1000
			oldDownTotal = newDownTotal
			oldUpTotal = newUpTotal
			fmt.Printf("downRate: %.2fMbps | upRate: %.2fMbps\n", downRate, upRate)
		}
	}()

	server, _ := CustomServer("http://shenzhen.cmcc.speedtest.shunshiidc.com:8080/speedtest/upload.php")
	//server, _ := CustomServer("http://192.168.5.237:8080/speedtest/upload.php")

	err := server.DownloadTest(false)
	if err != nil {
		fmt.Println("not found server")
		//t.Error(err)
	}

	GlobalDataManager.Wait()

	err = server.UploadTest(false)
	if err != nil {
		fmt.Println("not found server")
		//t.Error(err)
	}

	fmt.Printf(" \n")

	fmt.Printf("Download: %5.2f Mbit/s\n", server.DLSpeed)
	fmt.Printf("Upload: %5.2f Mbit/s\n\n", server.ULSpeed)
	valid := server.CheckResultValid()
	if !valid {
		fmt.Println("Warning: Result seems to be wrong. Please speedtest again.")
	}
}
