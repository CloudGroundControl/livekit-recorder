package recorder

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateFileSink(t *testing.T) {
	filename := "testing.txt"
	sink, err := NewFileSink(filename)
	require.NoError(t, err)
	require.NotNil(t, sink)
	require.Equal(t, filename, sink.Name())

	// Cleanup
	err = sink.Close()
	require.NoError(t, err)
	os.Remove(filename)
}

func TestWriteFileSink(t *testing.T) {
	filename := "testing.txt"
	sink, _ := NewFileSink(filename)
	defer func() {
		sink.Close()
		os.Remove(filename)
	}()

	n, err := sink.Write([]byte("Hello"))
	require.NoError(t, err)
	require.Equal(t, len("Hello"), n)
}

func TestWriteFileSinkWhenClosed(t *testing.T) {
	filename := "testing.txt"
	sink, _ := NewFileSink(filename)

	// Close file immediately
	err := sink.Close()
	require.NoError(t, err)

	// Try writing to the file
	n, err := sink.Write([]byte("Hello"))
	require.ErrorIs(t, err, ErrSinkClosed)
	require.Equal(t, 0, n)

	// Cleanup
	os.Remove(filename)
}

func TestCloseFileSinkMoreThanOnce(t *testing.T) {
	filename := "testing.txt"
	sink, _ := NewFileSink(filename)

	// Close file first time
	err := sink.Close()
	require.NoError(t, err)

	// Close file again
	err = sink.Close()
	require.ErrorIs(t, err, ErrSinkClosed)

	// Cleanup
	os.Remove(filename)
}

func TestCreateBufferSink(t *testing.T) {
	id := "testing"
	sink := NewBufferSink(id)
	require.NotNil(t, sink)
	require.Equal(t, id, sink.Name())

	err := sink.Close()
	require.NoError(t, err)
}

func TestWriteBufferSink(t *testing.T) {
	id := "testing"
	sink := NewBufferSink(id)
	defer sink.Close()

	p := []byte{'c', 'g', 'c'}
	n, err := sink.Write(p)
	require.NoError(t, err)
	require.Equal(t, 3, n)
}

func TestWriteBufferSinkWhenClosed(t *testing.T) {
	id := "testing"
	sink := NewBufferSink(id)
	sink.Close()

	p := []byte("Hello")
	n, err := sink.Write(p)
	require.ErrorIs(t, err, ErrSinkClosed)
	require.Equal(t, 0, n)
}

func TestReadBufferSink(t *testing.T) {
	id := "testing"
	sink := NewBufferSink(id)
	defer sink.Close()

	p := []byte{'c', 'g', 'c'}
	r := make([]byte, 3)
	sink.Write(p)
	n, err := sink.Read(r)
	require.NoError(t, err)
	require.Equal(t, 3, n)
	require.Equal(t, "cgc", string(r))
}

func TestCloseBufferSinkMoreThanOnce(t *testing.T) {
	id := "testing"
	sink := NewBufferSink(id)

	// Close once
	err := sink.Close()
	require.NoError(t, err)

	// Close again
	err = sink.Close()
	require.ErrorIs(t, err, ErrSinkClosed)
}
