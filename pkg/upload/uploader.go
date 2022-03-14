package upload

import "io"

type Uploader interface {
	// Key is a unique identifier for the file.
	Upload(key string, body io.Reader) error
	GetDirectory() string
}
