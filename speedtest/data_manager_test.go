package speedtest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func BenchmarkDataManager_NewDataChunk(b *testing.B) {
	dmp := NewDataManager()
	dmp.Reset()
	for i := 0; i < b.N; i++ {
		dmp.NewChunk()
	}
}

func BenchmarkDataManager_AddTotalDownload(b *testing.B) {
	dmp := NewDataManager()
	for i := 0; i < b.N; i++ {
		dmp.AddTotalDownload(43521)
	}
}

func TestDataManager_AddTotalDownload(t *testing.T) {
	dmp := NewDataManager()
	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 1000; j++ {
				dmp.AddTotalDownload(43521)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if dmp.download.GetTotalDataVolume() != 43521000000 {
		t.Fatal()
	}
}

func TestDataManager_GetAvgDownloadRate(t *testing.T) {
	dm := NewDataManager()
	dm.download.totalDataVolume = 3000000
	dm.captureTime = time.Second * 10

	result := dm.GetAvgDownloadRate()
	if result != 2.4 {
		t.Fatal()
	}
}

func TestDataManager_GetUploadConfirmationRatio(t *testing.T) {
	dm := NewDataManager()
	dm.upload.AddTotalReadVolume(2 * 1000 * 1000)
	dm.AddTotalUpload(500 * 1000)
	if got := dm.GetUploadBacklog(); got != 1500*1000 {
		t.Fatalf("got upload backlog %d, want %d", got, 1500*1000)
	}
	if got := dm.GetUploadConfirmationRatio(); got != 0.25 {
		t.Fatalf("got confirmation ratio %v, want 0.25", got)
	}

	dm.AddTotalUpload(3 * 1000 * 1000)
	if got := dm.GetUploadConfirmationRatio(); got != 1 {
		t.Fatalf("got confirmation ratio %v, want 1 after clamping", got)
	}
}

func TestDataManager_UploadWorkerLimit(t *testing.T) {
	dm := NewDataManager()
	dm.SetNThread(12)
	if dm.uploadMaxWorkers != 12 {
		t.Fatalf("explicit worker limit was capped at %d", dm.uploadMaxWorkers)
	}
	dm.SetNThread(0)
	if dm.uploadMaxWorkers != 8 {
		t.Fatalf("default worker limit is %d, want 8", dm.uploadMaxWorkers)
	}
}

func TestReducedUploadWorkerCount(t *testing.T) {
	if got := reducedUploadWorkerCount(10, 0.5, 1, 0); got != 5 {
		t.Fatalf("got %d workers after a 50%% rate drop, want 5", got)
	}
	if got := reducedUploadWorkerCount(10, 0.9, 1, 0); got != 9 {
		t.Fatalf("got %d workers after a mild rate drop, want 9", got)
	}
	if got := reducedUploadWorkerCount(2, 0.5, 1, 0); got != 1 {
		t.Fatalf("got %d workers at the minimum, want 1", got)
	}
}

func TestUploadRequestCountsOnlyConfirmedUploads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Errorf("failed to read upload: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New()
	target := &Server{URL: server.URL, Context: client}
	if err := uploadRequest(context.Background(), target, 0); err != nil {
		t.Fatalf("upload request failed: %v", err)
	}

	want := int64(ulSizes[0]*100-51) * 10
	if got := client.GetTotalUpload(); got != want {
		t.Fatalf("counted %d bytes, want %d", got, want)
	}
}

func TestUploadRequestDoesNotCountRejectedUploads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New()
	target := &Server{URL: server.URL, Context: client}
	if err := uploadRequest(context.Background(), target, 0); err == nil {
		t.Fatal("expected upload request to fail")
	}
	if got := client.GetTotalUpload(); got != 0 {
		t.Fatalf("counted %d bytes for a rejected upload", got)
	}
}

func TestDynamicRate(t *testing.T) {

	server, _ := CustomServer("http://shenzhen.cmcc.speedtest.shunshiidc.com:8080/speedtest/upload.php")
	//server, _ := CustomServer("http://192.168.5.237:8080/speedtest/upload.php")

	oldDownTotal := server.Context.GetTotalDownload()
	oldUpTotal := server.Context.GetTotalUpload()

	server.Context.SetRateCaptureFrequency(time.Millisecond * 100)
	server.Context.SetCaptureTime(time.Second)
	go func() {
		for i := 0; i < 2; i++ {
			time.Sleep(time.Second)
			newDownTotal := server.Context.GetTotalDownload()
			newUpTotal := server.Context.GetTotalUpload()

			downRate := float64(newDownTotal-oldDownTotal) * 8 / 1000 / 1000
			upRate := float64(newUpTotal-oldUpTotal) * 8 / 1000 / 1000
			oldDownTotal = newDownTotal
			oldUpTotal = newUpTotal
			fmt.Printf("downRate: %.2fMbps | upRate: %.2fMbps\n", downRate, upRate)
		}
	}()

	err := server.DownloadTest()
	if err != nil {
		fmt.Println("Warning: not found server")
		//t.Error(err)
	}

	server.Context.Wait()

	err = server.UploadTest()
	if err != nil {
		fmt.Println("Warning: not found server")
		//t.Error(err)
	}

	fmt.Printf(" \n")

	fmt.Printf("Download: %5.2f Mbit/s\n", server.DLSpeed)
	fmt.Printf("Upload: %5.2f Mbit/s\n\n", server.ULSpeed)
	valid := server.CheckResultValid()
	if !valid {
		fmt.Println("Warning: result seems to be wrong. Please test again.")
	}
}
