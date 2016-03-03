package io

import (
	"io"

	"github.com/v2ray/v2ray-core/common/alloc"
)

func RawReaderToChan(stream chan<- *alloc.Buffer, reader io.Reader) error {
	return ReaderToChan(stream, NewAdaptiveReader(reader))
}

// ReaderToChan dumps all content from a given reader to a chan by constantly reading it until EOF.
func ReaderToChan(stream chan<- *alloc.Buffer, reader Reader) error {
	for {
		buffer, err := reader.Read()
		if buffer.Len() > 0 {
			stream <- buffer
		} else {
			buffer.Release()
		}

		if err != nil {
			return err
		}
	}
}

func ChanToRawWriter(writer io.Writer, stream <-chan *alloc.Buffer) error {
	return ChanToWriter(NewAdaptiveWriter(writer), stream)
}

// ChanToWriter dumps all content from a given chan to a writer until the chan is closed.
func ChanToWriter(writer Writer, stream <-chan *alloc.Buffer) error {
	for buffer := range stream {
		err := writer.Write(buffer)
		buffer.Release()
		if err != nil {
			return err
		}
	}
	return nil
}
