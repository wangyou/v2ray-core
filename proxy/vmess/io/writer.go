package io

import (
	"hash/fnv"

	"github.com/v2ray/v2ray-core/common/alloc"
	v2io "github.com/v2ray/v2ray-core/common/io"
	"github.com/v2ray/v2ray-core/common/serial"
)

type AuthChunkWriter struct {
	writer v2io.Writer
}

func NewAuthChunkWriter(writer v2io.Writer) *AuthChunkWriter {
	return &AuthChunkWriter{
		writer: writer,
	}
}

func (this *AuthChunkWriter) Write(buffer *alloc.Buffer) error {
	Authenticate(buffer)
	return this.writer.Write(buffer)
}

func Authenticate(buffer *alloc.Buffer) {
	fnvHash := fnv.New32a()
	fnvHash.Write(buffer.Value)

	buffer.SliceBack(4)
	fnvHash.Sum(buffer.Value[:0])

	buffer.Prepend(serial.Uint16Literal(uint16(buffer.Len())).Bytes())
}
