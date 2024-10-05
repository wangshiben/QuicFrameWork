package Writer

import (
	"bytes"
	"net/http"
)

type Writer struct {
	writer http.ResponseWriter
	buffer bytes.Buffer
}

func (w *Writer) Header() http.Header {
	return w.writer.Header()
}

func (w *Writer) WriteHeader(statusCode int) {
	w.writer.WriteHeader(statusCode)
}

func (w *Writer) Write(data []byte) (int, error) {
	return w.buffer.Write(data)
}

func (w *Writer) FinishWrite() (int64, error) {
	return w.buffer.WriteTo(w.writer)
}
func NewWriter(w http.ResponseWriter) *Writer {
	return &Writer{
		writer: w,
		buffer: bytes.Buffer{},
	}
}
