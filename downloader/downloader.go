package downloader

import (
	"ImageCrawler/models"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"net/url"
	"sync"
)

func DownloadImages(pageURL string) ([]models.ImageBlob, error) {
	imgURLs, _ := PageImageURLs(pageURL)

	var imageBlobs []models.ImageBlob
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, imgURL := range imgURLs {
		wg.Add(1)
		go func(imgURL string) {
			defer wg.Done()

			imgResp, err := http.Get(imgURL)
			if err != nil {
				fmt.Printf("failed to download image: %v\n", err)
				return
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					return
				}
			}(imgResp.Body)

			if imgResp.StatusCode != http.StatusOK {
				fmt.Printf("error fetching image: %v\n", imgResp.Status)
				return
			}

			imgData, err := io.ReadAll(imgResp.Body)
			if err != nil {
				fmt.Printf("failed to read image: %v\n", err)
				return
			}

			mu.Lock()
			imageBlobs = append(imageBlobs, models.ImageBlob{
				URL:  imgURL,
				Data: imgData,
			})
			mu.Unlock()
		}(imgURL.URL)
	}
	wg.Wait()

	return imageBlobs, nil
}

func PageImageURLs(pageURL string) ([]models.ImageUrl, error) {
	var imageUrls []models.ImageUrl

	resp, err := http.Get(pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching page: %v", resp.Status)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	baseURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %v", err)
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			for _, a := range n.Attr {
				if a.Key == "src" {
					imgURL, err := baseURL.Parse(a.Val)
					if err == nil {
						imageUrls = append(imageUrls, models.ImageUrl{URL: imgURL.String(), Host: imgURL.Host, Path: imgURL.Path})
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return imageUrls, nil
}
