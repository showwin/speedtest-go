package main

import (
	"context"
	"fmt"
	"github.com/showwin/speedtest-go/speedtest"
	"time"
)

func main() {
	server, err := speedtest.CustomServer("http://speedtest.139play.com:8080/speedtest/upload.php")
	if err != nil {
		fmt.Println(err)
		return
	}
	//timeoutContext, cancel := context.WithTimeout(context.Background(), time.Second*10)
	//defer cancel()
	//_, err = server.TCPPing(timeoutContext, 5, time.Second, func(latency time.Duration) {
	//	fmt.Println(latency)
	//})
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}

	fmt.Println()
	//timeoutContext1, cancel1 := context.WithTimeout(context.Background(), time.Second*10)
	//defer cancel1()
	_, err = server.HTTPPing(context.TODO(), 5, time.Second, func(latency time.Duration) {
		fmt.Println(latency * 2)
		//fmt.Println(latency)
	})
	if err != nil {
		fmt.Println(err)
		return
	}
}
