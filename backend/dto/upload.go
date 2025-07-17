package dto

type UploadResponse struct {
	ID       string `json:"id" example:"abc123"`
	URL      string `json:"url" example:"/uploads/models/abc.jpg"`
	Usage    string `json:"usage" example:"avatar"`
	MimeType string `json:"mimeType" example:"image/jpeg"`
	Size     int64  `json:"size" example:"204800"`
}
