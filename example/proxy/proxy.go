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

	span, err := infector.ParseSpanFromCtx(ctx)
	if err == nil {
		defer span.Cancel()
		header = span.GetHttpHeader()
	}
	log.Println(err)

	// http get
	resp, err := req.Get(url, header)
	if err != nil {
		log.Println(resp, err)
		return
	}

	c.Writer.Write(resp.Bytes())
	c.Status(resp.Response().StatusCode)
}

func main() {
	server()
}
