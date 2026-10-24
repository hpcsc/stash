package item

type CaptureRequest struct {
	Body      string `json:"body" validate:"required"`
	SourceApp string `json:"sourceApp"`
	SourceURL string `json:"sourceUrl"`
}
