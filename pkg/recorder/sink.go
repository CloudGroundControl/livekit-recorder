package recorder

import (
	"errors"
	"os"

	"github.com/pion/transport/packetio"
)

type sinkState string

const (
	sinkAvailable sinkState = "available"
	sinkClosed    sinkState = "closed"
)

var ErrSinkClosed = errors.New("sink closed")

type Sink interface {
	Name() string
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
}

type fileSink struct {
	file  *os.File
	state sinkState
}

func NewFileSink(filename string) (Sink, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return &fileSink{file, sinkAvailable}, nil
}

func (s *fileSink) Name() string {
	return s.file.Name()
}

func (s *fileSink) Read(p []byte) (int, error) {
	return s.file.Read(p)
}

func (s *fileSink) Write(p []byte) (int, error) {
	if s.state == sinkClosed {
		return 0, ErrSinkClosed
	}
	return s.file.Write(p)
}

func (s *fileSink) Close() error {
	if s.state == sinkClosed {
		return ErrSinkClosed
	}
	s.state = sinkClosed
	return s.file.Close()
}

type bufferSink struct {
	id     string
	buffer *packetio.Buffer
	state  sinkState
}

func NewBufferSink(id string) Sink {
	buffer := packetio.NewBuffer()
	return &bufferSink{id, buffer, sinkAvailable}
}

func (s *bufferSink) Name() string {
	return s.id
}

func (s *bufferSink) Read(p []byte) (int, error) {
	return s.buffer.Read(p)
}

func (s *bufferSink) Write(p []byte) (int, error) {
	if s.state == sinkClosed {
		return 0, ErrSinkClosed
	}
	return s.buffer.Write(p)
}

func (s *bufferSink) Close() error {
	if s.state == sinkClosed {
		return ErrSinkClosed
	}
	s.state = sinkClosed
	return s.buffer.Close()
}
