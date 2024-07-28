package handlers

import (
	"ImageCrawler/cache"
	"ImageCrawler/downloader"
	"ImageCrawler/models"
	"ImageCrawler/s3client"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
)

var s3Client = s3client.NewS3Client()
var urlCache = cache.NewRedisCache()

// CheckImages handles GET requests to check if images by URL already exist in S3
func CheckImages(c *gin.Context) {
	pageURL := c.Query("url")
	if pageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL query parameter is required"})
		return
	}

	// Check the cache first
	metadata, found := urlCache.Get(pageURL)
	if !found {
		c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
		return
	}

	var imageNames []string
	for _, img := range metadata.Images {
		imageNames = append(imageNames, img.Key)
	}
	c.JSON(http.StatusOK, gin.H{"images": imageNames})
}

// ProcessURL handles POST requests to process a new URL and upload images to S3
func ProcessURL(c *gin.Context) {
	var req models.URLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if urlCache.Exists(req.URL) {
		c.JSON(http.StatusConflict, gin.H{"error": "URL already processed"})
		return
	}

	imageBlobs, err := downloader.DownloadImages(req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download images ${er}"})
		return
	}

	metadata := models.Metadata{URL: req.URL}
	for _, img := range imageBlobs {
		key := generateFileKey(img.URL)
		if err := s3Client.PutObject(key, img.Data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image to S3"})
			return
		}
		metadata.Images = append(metadata.Images, models.Image{Key: key, URL: img.URL})
	}

	urlCache.Set(req.URL, metadata)
	c.JSON(http.StatusOK, gin.H{"message": "Images uploaded successfully"})
}

// UpdateURL handles PUT requests to update an existing URL or create new if it doesn't exist
func UpdateURL(c *gin.Context) {
	var req models.URLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var imgToDelete []string

	if urlCache.Exists(req.URL) {
		pageImageUrls, err := downloader.PageImageURLs(req.URL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get images"})
			return
		}

		metadata, found := urlCache.Get(req.URL)
		if !found {
			c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
			return
		}

		for _, metadataEntity := range metadata.Images {
			if !contains(metadataEntity.URL, pageImageUrls) {
				imgToDelete = append(imgToDelete, metadataEntity.Key)
			}
		}
	}

	for _, imgtoDel := range imgToDelete {
		if err := s3Client.RemoveObject(imgtoDel); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete old images from S3"})
			return
		}
	}

	imageBlobs, err := downloader.DownloadImages(req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download images"})
		return
	}

	metadata := models.Metadata{URL: req.URL}

	for _, img := range imageBlobs {
		key := generateFileKey(img.URL)
		if err := s3Client.PutObject(key, img.Data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image to S3"})
			return
		}
		metadata.Images = append(metadata.Images, models.Image{Key: key, URL: img.URL})
	}

	urlCache.Set(req.URL, metadata)
	c.JSON(http.StatusOK, gin.H{"message": "Images updated successfully"})
}

func contains(imgUrl string, imageArr []models.ImageUrl) bool {
	for _, i := range imageArr {
		if i.URL == imgUrl {
			return true
		}
	}
	return false
}

func generateFileKey(imgURL string) string {
	iurl, err := url.Parse(imgURL)
	if err != nil {
		log.Fatal(err)
	}
	hostname := strings.TrimPrefix(iurl.Hostname(), "www.")
	return fmt.Sprintf("images/%s/%s", deterministicGUID(hostname), path.Base(imgURL))
}

func deterministicGUID(pageUrl string) string {
	md5hash := md5.New()
	md5hash.Write([]byte(pageUrl))
	md5string := hex.EncodeToString(md5hash.Sum(nil))

	hostUuid, err := uuid.FromBytes([]byte(md5string[0:16]))
	if err != nil {
		log.Fatal(err)
	}
	return hostUuid.String()
}
