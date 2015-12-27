package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"io"
)

func NewAesDecryptionStream(key []byte, iv []byte) (cipher.Stream, error) {
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewCFBDecrypter(aesBlock, iv), nil
}

func NewAesEncryptionStream(key []byte, iv []byte) (cipher.Stream, error) {
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewCFBEncrypter(aesBlock, iv), nil
}

type cryptionReader struct {
	stream cipher.Stream
	reader io.Reader
}

func NewCryptionReader(stream cipher.Stream, reader io.Reader) io.Reader {
	return &cryptionReader{
		stream: stream,
		reader: reader,
	}
}

func (this *cryptionReader) Read(data []byte) (int, error) {
	nBytes, err := this.reader.Read(data)
	if nBytes > 0 {
		this.stream.XORKeyStream(data[:nBytes], data[:nBytes])
	}
	return nBytes, err
}

type cryptionWriter struct {
	stream cipher.Stream
	writer io.Writer
}

func NewCryptionWriter(stream cipher.Stream, writer io.Writer) io.Writer {
	return &cryptionWriter{
		stream: stream,
		writer: writer,
	}
}

func (this *cryptionWriter) Write(data []byte) (int, error) {
	this.stream.XORKeyStream(data, data)
	return this.writer.Write(data)
}
