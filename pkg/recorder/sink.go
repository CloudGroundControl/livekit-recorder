package recorder

import "os"

type RecorderSink interface {
	Name() string
	Write([]byte) (int, error)
	Close() error
}

func NewFileSink(filename string) (RecorderSink, error) {
	return os.Create(filename)
}
