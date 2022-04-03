package recorder

import (
	"bufio"
	"os"

	"github.com/labstack/gommon/log"
	"github.com/pion/transport/packetio"
)

type Sink interface {
	Name() string
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
}

type fileSink struct {
	bw   *bufio.Writer
	file *os.File
	name string
}

func NewFileSink(filename string) (Sink, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	bw := bufio.NewWriter(f)
	return &fileSink{bw, f, filename}, nil
}

func (s *fileSink) Name() string {
	return s.name
}

func (s *fileSink) Read(b []byte) (int, error) {
	return s.file.Read(b)
}

func (s *fileSink) Write(b []byte) (int, error) {
	return s.bw.Write(b)
}

func (s *fileSink) Close() error {
	err := s.bw.Flush()
	if err != nil {
		log.Errorf("cannot flush file sink | error: %v", err)
	}
	return s.file.Close()
}

type bufferSink struct {
	id     string
	buffer *packetio.Buffer
}

func NewBufferSink(id string) Sink {
	buffer := packetio.NewBuffer()
	return &bufferSink{id, buffer}
}

func (s *bufferSink) Name() string {
	return s.id
}

func (s *bufferSink) Read(p []byte) (int, error) {
	return s.buffer.Read(p)
}

func (s *bufferSink) Write(p []byte) (int, error) {
	return s.buffer.Write(p)
}

func (s *bufferSink) Close() error {
	return s.buffer.Close()
}
