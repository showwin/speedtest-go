package speedtest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"
)

// confirmedUpload returns the upload bytes confirmed by successful upload responses, which
// GetTotalUpload reports only where the platform does not report acknowledged bytes.
func confirmedUpload(s *Speedtest) int64 {
	return s.Manager.(*DataManager).upload.GetTotalDataVolume()
}

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

func TestDataManager_FinalRateStopsBeforeCancellation(t *testing.T) {
	const (
		captureTime = 200 * time.Millisecond
		perRequest  = int64(1_000) // one request a millisecond: about 1 MB/s
		afterStop   = int64(1_000_000)
	)
	dm := NewDataManager()
	dm.SetNThread(1)
	dm.SetCaptureTime(captureTime)
	dm.SetRateCaptureFrequency(5 * time.Millisecond)

	var direction *TestDirection
	direction = dm.RegisterDownloadHandler(func() {
		direction.AddTotalDataVolume(perRequest)
		time.Sleep(time.Millisecond)
	})
	direction.Start(func() {
		direction.AddTotalDataVolume(afterStop)
	}, 0)

	got := direction.finalRate()
	if got <= 0 {
		t.Fatalf("final rate is %v, want a positive value", got)
	}
	// The cancellation bytes alone would add more than 7 MB/s to the last two thirds of the test.
	if limit := 2 * float64(perRequest) / time.Millisecond.Seconds(); got > limit {
		t.Fatalf("final rate is %v bytes/s, which includes bytes added after the stop boundary", got)
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

func TestDataManager_ConnectionLimit(t *testing.T) {
	dm := NewDataManager()
	dm.SetNThread(12)
	if dm.nThread != 12 || dm.nUploadThread != 12 {
		t.Fatalf("explicit connection limits are %d and %d, want 12", dm.nThread, dm.nUploadThread)
	}
	dm.SetNThread(0)
	if want := runtime.NumCPU(); dm.nThread != want {
		t.Fatalf("default download connections are %d, want %d", dm.nThread, want)
	}
	if dm.nUploadThread != 8 {
		t.Fatalf("default upload connections are %d, want 8", dm.nUploadThread)
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
	if got := confirmedUpload(client); got != want {
		t.Fatalf("counted %d bytes, want %d", got, want)
	}
}

func TestResolveUploadURLFollowsTemporaryRedirect(t *testing.T) {
	var received int64
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodPost {
			t.Errorf("request method is %s, want POST", r.Method)
		}
		var err error
		received, err = io.Copy(io.Discard, r.Body)
		if err != nil {
			t.Errorf("failed to read redirected upload: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetServer.URL, http.StatusTemporaryRedirect)
	}))
	defer redirectServer.Close()

	client := New()
	target := &Server{URL: redirectServer.URL, Context: client}
	target.resolveUploadURL(context.Background())
	if target.URL != targetServer.URL {
		t.Fatalf("resolved upload URL is %q, want %q", target.URL, targetServer.URL)
	}
	if err := uploadRequest(context.Background(), target, 0); err != nil {
		t.Fatalf("upload request failed: %v", err)
	}

	want := int64(ulSizes[0]*100-51) * 10
	if received != want {
		t.Fatalf("resolved endpoint received %d bytes, want %d", received, want)
	}
	if got := confirmedUpload(client); got != want {
		t.Fatalf("counted %d upload bytes, want %d", got, want)
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
	if got := confirmedUpload(client); got != 0 {
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
