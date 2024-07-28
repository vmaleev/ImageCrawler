package models

type ImageBlob struct {
	URL  string
	Data []byte
}

type ImageUrl struct {
	URL  string
	Host string
	Path string
}

type URLRequest struct {
	URL string `json:"url" binding:"required"`
}

type Metadata struct {
	URL    string  `json:"url"`
	Images []Image `json:"images"`
}

type Image struct {
	Key string `json:"key"`
	URL string `json:"url"`
}
