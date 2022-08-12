package main

import (
	"encoding/hex"
	"fmt"
	"io"
)

type loggingReader struct {
	r   io.Reader
	out io.Writer
}

func (l loggingReader) Close() error {
	if rc, ok := l.r.(io.Closer); ok {
		return rc.Close()
	}

	return nil
}
func (l loggingReader) Read(p []byte) (n int, err error) {
	n, err = l.r.Read(p)
	if n > 0 {
		fmt.Fprintf(l.out, "out >>>>> %d\n", n)
		fmt.Println(hex.EncodeToString(p[:n]))
		fmt.Fprintln(l.out, hex.Dump(p[:n]))
	}
	if err != nil {
		fmt.Fprintf(l.out, "out error %v", err)
	}

	return
}

func NewLoggingReader(r io.Reader, out io.Writer) io.ReadCloser {
	return loggingReader{
		r:   r,
		out: out,
	}
}
