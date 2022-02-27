package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imroc/req"
	"github.com/rfyiamcool/go-infector"
)

var (
	url = "http://127.0.0.1:8282/user"
)

func server() {
	r := gin.Default()
	r.Use(infector.GinMiddleware())
	r.GET("/proxy", handleProxy)
	r.Run(":8181")
}

func handleProxy(c *gin.Context) {
	// make delay case
	time.Sleep(3 * time.Second)

	// span
	var (
		header = http.Header{}
		ctx    = c.Request.Context()
	)

	// set custom header
	header.Set("key-proxy", "xxx")

	span, err := infector.ParseSpanFromCtx(ctx)
	if err == nil { // don't inject span in middleware.
		defer span.Cancel()
		header = span.GetHttpHeader(header)
	} else {
		log.Println(err.Error())
	}

	var retryCount = 3
	for i := 0; i < retryCount; i++ {
		if !span.IsRetryON() {
			break
		}

		// http get
		resp, err := req.Get(url, header)
		if err != nil {
			log.Println(resp, err)
			return
		}

		c.Writer.Write(resp.Bytes())
		c.Status(resp.Response().StatusCode)
	}
}

func main() {
	server()
}
