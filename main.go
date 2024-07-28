package main

import (
	"ImageCrawler/handlers"
	"github.com/fufuok/favicon"
	"github.com/gin-gonic/gin"
	"github.com/penglongli/gin-metrics/ginmetrics"
)

func main() {
	var favData []byte
	r := gin.Default()

	m := ginmetrics.GetMonitor()
	m.SetMetricPath("/metrics")
	m.Use(r)

	r.Use(favicon.New(favicon.Config{
		FileData: favData,
	}))

	r.GET("/images", handlers.CheckImages)
	r.POST("/images", handlers.ProcessURL)
	r.PUT("/images", handlers.UpdateURL)

	err := r.Run(":8080")
	if err != nil {
		return
	}
}
