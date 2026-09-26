package Writer

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
)

type Writer struct {
	writer     http.ResponseWriter
	buffer     bytes.Buffer
	statusCode int
	committed  bool
}

func (w *Writer) Header() http.Header {
	return w.writer.Header()
}

func (w *Writer) WriteHeader(statusCode int) {
	if w.statusCode != 0 || w.committed {
		return
	}
	w.statusCode = statusCode
}

func (w *Writer) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.buffer.Write(data)
}

func (w *Writer) FinishWrite() (int64, error) {
	w.commitHeader()
	return w.buffer.WriteTo(w.writer)
}
func (w *Writer) Reset() bool {
	w.buffer.Reset()
	if w.committed {
		return false
	}
	w.statusCode = 0
	return true
}
func (w *Writer) commitHeader() {
	if w.committed {
		return
	}
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	w.writer.WriteHeader(w.statusCode)
	w.committed = true
}
func (w *Writer) Flush() {
	flusher, ok := w.writer.(http.Flusher)
	if !ok {
		return
	}
	w.commitHeader()
	to, err := w.buffer.WriteTo(w.writer)
	if err != nil {
		fmt.Printf("Error writing to response: %v", err)
		fmt.Printf("Written bytes: %d", to)
	}
	flusher.Flush()
}
func (w *Writer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.writer.(http.Hijacker).Hijack()
}
func NewWriter(w http.ResponseWriter) *Writer {
	return &Writer{
		writer: w,
		buffer: bytes.Buffer{},
	}
}
