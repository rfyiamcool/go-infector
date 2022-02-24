package main

import (
	"context"
	"log"
	"time"

	"github.com/imroc/req"
	"github.com/rfyiamcool/go-infector"
)

var (
	url = "http://127.0.0.1:8181/proxy"
)

func handleRequest() {
	start := time.Now()

	span := infector.NewSpanContext(context.Background(), 2*time.Second, infector.RetryOff)
	defer span.Cancel()

	header := span.GetHttpHeader()
	header.Set("custom-key1", "custom-val1")
	resp, err := req.Get(url, header)

	log.Println(resp.String(), err)
	log.Println(time.Since(start).String())
}

func handleGinMiddleware() {
	start := time.Now()

	span := infector.NewSpanContext(context.Background(), 1*time.Second, infector.RetryOff)
	defer span.Cancel()

	header := span.GetHttpHeader()
	header.Set("custom-key1", "custom-val1")

	time.Sleep(2 * time.Second) // make delay case
	resp, err := req.Get(url, header)

	log.Println(resp.String(), err)
	log.Println(time.Since(start).String())
}

func main() {
	handleRequest()
}
