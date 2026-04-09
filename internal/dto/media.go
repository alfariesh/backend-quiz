package dto

import "io"

type UploadMediaRequest struct {
	FileName    string
	FileSize    int
	ContentType string
	Body        io.Reader
}
