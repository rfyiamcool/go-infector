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

	// span
	span := infector.NewSpanContext(context.Background(), 2*time.Second, infector.RetryOff)
	defer span.Cancel()

	// request
	header := span.GetHttpHeader()           // inject infector args to header, then return the header
	header.Set("custom-key1", "custom-val1") // set custom kv in header.
	resp, err := req.Get(url, header)
	log.Println(resp.String(), err)

	log.Println(time.Since(start).String())
}

func main() {
	handleRequest()
}
