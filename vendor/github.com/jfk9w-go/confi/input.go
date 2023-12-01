package confi

import (
	"io"
	"os"
)

type Input interface {
	Reader() (io.Reader, error)
}

type File string

func (f File) Path() string               { return string(f) }
func (f File) Reader() (io.Reader, error) { return os.Open(f.Path()) }

type Reader struct {
	R io.Reader
}

func (r Reader) Reader() (io.Reader, error) { return r.R, nil }

func CloseQuietly(value any) {
	if value == os.Stdin || value == os.Stdout || value == os.Stderr {
		return
	}

	if closer, ok := value.(io.Closer); ok {
		_ = closer.Close()
	}
}
