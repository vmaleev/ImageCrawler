package handlers

import (
	"ImageCrawler/models"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDeterministicGUIDConsistency(t *testing.T) {
	host := "example.com"
	id1, err := deterministicGUID(host)
	if err != nil {
		t.Fatalf("deterministicGUID returned error: %v", err)
	}
	id2, err := deterministicGUID(host)
	if err != nil {
		t.Fatalf("deterministicGUID returned error: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("expected identical GUIDs, got %s and %s", id1, id2)
	}
	if len(id1) == 0 {
		t.Fatalf("GUID should not be empty")
	}
}

func TestGenerateFileKeyInvalidURLReturnsError(t *testing.T) {
	if _, err := generateFileKey("https://example.com/\x00.jpg"); err == nil {
		t.Fatalf("expected generateFileKey to return an error for invalid URL")
	}
}

func TestProcessURLInvalidImageURLReturnsServerError(t *testing.T) {
	restore := stubDownloadImages(func(string) ([]models.ImageBlob, error) {
		return []models.ImageBlob{{URL: "https://example.com/\x00.jpg", Data: []byte("image-data")}}, nil
	})
	defer restore()

	w := performJSONRequest(http.MethodPost, "/process", `{"url":"https://page.example"}`, ProcessURL)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
	if body := w.Body.String(); body != "{\"error\":\""+generateImageKeyErrorMessage+"\"}" {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestUpdateURLInvalidImageURLReturnsServerError(t *testing.T) {
	restore := stubDownloadImages(func(string) ([]models.ImageBlob, error) {
		return []models.ImageBlob{{URL: "https://example.com/\x00.jpg", Data: []byte("image-data")}}, nil
	})
	defer restore()

	w := performJSONRequest(http.MethodPut, "/update", `{"url":"https://page.example"}`, UpdateURL)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
	if body := w.Body.String(); body != "{\"error\":\""+generateImageKeyErrorMessage+"\"}" {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func stubDownloadImages(fn func(string) ([]models.ImageBlob, error)) func() {
	original := downloadImages
	downloadImages = fn
	return func() {
		downloadImages = original
	}
}

func performJSONRequest(method, target, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)
	return w
}
