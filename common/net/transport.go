package net

import (
	"io"

	"github.com/v2ray/v2ray-core/common/alloc"
)

// ReadFrom reads from a reader and put all content to a buffer.
// If buffer is nil, ReadFrom creates a new normal buffer.
func ReadFrom(reader io.Reader, buffer *alloc.Buffer) (*alloc.Buffer, error) {
	if buffer == nil {
		buffer = alloc.NewBuffer()
	}
	nBytes, err := reader.Read(buffer.Value)
	buffer.Slice(0, nBytes)
	return buffer, err
}

// ReadAllBytes reads all bytes required from reader, if no error happens.
func ReadAllBytes(reader io.Reader, buffer []byte) (int, error) {
	bytesRead := 0
	bytesAsked := len(buffer)
	for bytesRead < bytesAsked {
		nBytes, err := reader.Read(buffer[bytesRead:])
		bytesRead += nBytes
		if err != nil {
			return bytesRead, err
		}
	}
	return bytesRead, nil
}

// ReaderToChan dumps all content from a given reader to a chan by constantly reading it until EOF.
func ReaderToChan(stream chan<- *alloc.Buffer, reader io.Reader) error {
	allocate := alloc.NewBuffer
	large := false
	for {
		buffer, err := ReadFrom(reader, allocate())
		if buffer.Len() > 0 {
			stream <- buffer
		} else {
			buffer.Release()
		}
		if err != nil {
			return err
		}
		if buffer.IsFull() && !large {
			allocate = alloc.NewLargeBuffer
			large = true
		} else if !buffer.IsFull() {
			allocate = alloc.NewBuffer
			large = false
		}
	}
}

// ChanToWriter dumps all content from a given chan to a writer until the chan is closed.
func ChanToWriter(writer io.Writer, stream <-chan *alloc.Buffer) error {
	for buffer := range stream {
		nBytes, err := writer.Write(buffer.Value)
		if nBytes < buffer.Len() {
			_, err = writer.Write(buffer.Value[nBytes:])
		}
		buffer.Release()
		if err != nil {
			return err
		}
	}
	return nil
}
