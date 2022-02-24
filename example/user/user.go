package main

import (
	"github.com/gin-gonic/gin"
	"github.com/rfyiamcool/go-infector"
)

func server() {
	r := gin.Default()
	r.Use(infector.GinMiddleware())
	r.GET("/user", handleUser)
	r.Run(":8282")
}

func handleUser(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "resp in user",
	})
}

func main() {
	server()
}
