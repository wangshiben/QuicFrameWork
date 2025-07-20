package Writer

import (
	"bytes"
	"fmt"
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
	w.Flush()
}

func (w *Writer) Write(data []byte) (int, error) {
	return w.buffer.Write(data)
}

func (w *Writer) FinishWrite() (int64, error) {
	return w.buffer.WriteTo(w.writer)
}
func (w *Writer) Flush() {
	flusher, ok := w.writer.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	//if w.buffer.Len() != 0 {
	to, err := w.buffer.WriteTo(w.writer)
	if err != nil {
		fmt.Printf("Error writing to response: %v", err)
		fmt.Printf("Written bytes: %d", to)
	}
	//}

	flusher.Flush()
}
func NewWriter(w http.ResponseWriter) *Writer {
	return &Writer{
		writer: w,
		buffer: bytes.Buffer{},
	}
}
