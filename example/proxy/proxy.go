package main

import (
	"log"
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
	select {
	case <-c.Request.Context().Done():
	case <-time.After(10 * time.Second):
	}

	time.Sleep(1 * time.Second)

	span, _ := infector.ParseSpanFromCtx(c.Request.Context())
	defer span.Cancel()

	header := span.GetHttpHeader()
	resp, err := req.Get(url, header)
	log.Println(resp, err)

	c.JSON(200, gin.H{
		"message": "resp in user",
	})
}

func main() {
	server()
}
