package handlers

import (
	"ImageCrawler/models"
	"ImageCrawler/safeurl"
	"crypto/md5"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Cache interface {
	Get(key string) (models.Metadata, bool)
	Set(key string, val models.Metadata)
	Exists(key string) bool
	Invalidate(key string)
}

type Storage interface {
	PutObject(key string, data []byte) error
	RemoveObject(key string) error
}

type DownloadImagesFunc func(pageURL string) ([]models.ImageBlob, error)

type PageImageURLsFunc func(pageURL string) ([]models.ImageUrl, error)

type Handler struct {
	storage        Storage
	cache          Cache
	downloadImages DownloadImagesFunc
	pageImageURLs  PageImageURLsFunc
}

func New(storage Storage, cache Cache, downloadImages DownloadImagesFunc, pageImageURLs PageImageURLsFunc) *Handler {
	return &Handler{
		storage:        storage,
		cache:          cache,
		downloadImages: downloadImages,
		pageImageURLs:  pageImageURLs,
	}
}

const generateImageKeyErrorMessage = "Failed to generate image key"

// CheckImages handles GET requests to check if images by URL already exist in S3
func (h *Handler) CheckImages(c *gin.Context) {
	pageURL := c.Query("url")
	if pageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL query parameter is required"})
		return
	}

	// Check the cache first
	metadata, found := h.cache.Get(pageURL)
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
func (h *Handler) ProcessURL(c *gin.Context) {
	var req models.URLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := safeurl.ValidateURL(req.URL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid URL: %v", err)})
		return
	}

	if h.cache.Exists(req.URL) {
		c.JSON(http.StatusConflict, gin.H{"error": "URL already processed"})
		return
	}

	imageBlobs, err := h.downloadImages(req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to download images: %v", err)})
		return
	}

	metadata := models.Metadata{URL: req.URL}
	for _, img := range imageBlobs {
		key, err := generateFileKey(img.URL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": generateImageKeyErrorMessage})
			return
		}
		if err := h.storage.PutObject(key, img.Data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image to S3"})
			return
		}
		metadata.Images = append(metadata.Images, models.Image{Key: key, URL: img.URL})
	}

	h.cache.Set(req.URL, metadata)
	c.JSON(http.StatusOK, gin.H{"message": "Images uploaded successfully"})
}

// UpdateURL handles PUT requests to update an existing URL or create new if it doesn't exist
func (h *Handler) UpdateURL(c *gin.Context) {
	var req models.URLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := safeurl.ValidateURL(req.URL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid URL: %v", err)})
		return
	}

	var imgToDelete []string

	if h.cache.Exists(req.URL) {
		pageImageUrls, err := h.pageImageURLs(req.URL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get images"})
			return
		}

		metadata, found := h.cache.Get(req.URL)
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
		if err := h.storage.RemoveObject(imgtoDel); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete old images from S3"})
			return
		}
	}

	imageBlobs, err := h.downloadImages(req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download images"})
		return
	}

	metadata := models.Metadata{URL: req.URL}

	for _, img := range imageBlobs {
		key, err := generateFileKey(img.URL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": generateImageKeyErrorMessage})
			return
		}
		if err := h.storage.PutObject(key, img.Data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image to S3"})
			return
		}
		metadata.Images = append(metadata.Images, models.Image{Key: key, URL: img.URL})
	}

	h.cache.Set(req.URL, metadata)
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

func generateFileKey(imgURL string) (string, error) {
	iurl, err := url.Parse(imgURL)
	if err != nil {
		return "", err
	}
	hostname := strings.TrimPrefix(iurl.Hostname(), "www.")
	guid := deterministicGUID(hostname)
	return fmt.Sprintf("images/%s/%s", guid, path.Base(imgURL)), nil
}

func deterministicGUID(hostname string) string {
	hash := md5.Sum([]byte(hostname))
	return uuid.Must(uuid.FromBytes(hash[:])).String()
}
