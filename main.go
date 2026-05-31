package main

import (
	"ImageCrawler/cache"
	"ImageCrawler/downloader"
	"ImageCrawler/handlers"
	"ImageCrawler/s3client"
	"log"

	"github.com/fufuok/favicon"
	"github.com/gin-gonic/gin"
	"github.com/penglongli/gin-metrics/ginmetrics"
)

func main() {
	s3Client, err := s3client.NewS3Client()
	if err != nil {
		log.Fatalf("failed to initialize S3 client: %v", err)
	}

	urlCache, err := cache.NewRedisCache()
	if err != nil {
		log.Fatalf("failed to initialize Redis cache: %v", err)
	}

	h := handlers.New(s3Client, urlCache, downloader.DownloadImages, downloader.PageImageURLs)

	var favData []byte
	r := gin.Default()

	m := ginmetrics.GetMonitor()
	m.SetMetricPath("/metrics")
	m.Use(r)

	r.Use(favicon.New(favicon.Config{
		FileData: favData,
	}))

	r.GET("/images", h.CheckImages)
	r.POST("/images", h.ProcessURL)
	r.PUT("/images", h.UpdateURL)

	err = r.Run(":8080")
	if err != nil {
		return
	}
}
