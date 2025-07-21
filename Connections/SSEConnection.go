package Connections

import (
	"github.com/wangshiben/QuicFrameWork/RequestX"
	"github.com/wangshiben/QuicFrameWork/consts"
	"net/http"
	"strings"
)

const Close = consts.Close

type SSEConnection struct {
	Request     RequestX.Request // ParseFrom  HttpRequest(for session or some path Param)
	waitChannel chan string
	writer      http.ResponseWriter // http2 writer with flush interface
	flusher     http.Flusher
	isClosed    bool
}
type SSEEvent struct {
	Event string
	Data  string
	Id    string
}

func (s *SSEEvent) parse() []byte {
	var b strings.Builder

	if len(s.Event) != 0 {
		b.WriteString("event: ")
		b.WriteString(escape(s.Event))
		b.WriteString("\n")
	}
	if len(s.Id) != 0 {
		b.WriteString("id: ")
		b.WriteString(escape(s.Id))
		b.WriteString("\n")
	}
	b.WriteString("data: ")
	b.WriteString(escape(s.Data))
	b.WriteString("\n\n")

	return []byte(b.String())
}

// escape prevent user use \n in their code to break the SSE protocol
func escape(s string) string {
	return strings.ReplaceAll(s, "\n", "\\n")
}

// SendEvent send a SSEEvent to the client
func (s *SSEConnection) SendEvent(event *SSEEvent) error {
	_, err := s.Write(event.parse())
	if err != nil {
		return err
	}
	return nil
}

// Write for user to write origin bytes to the client
// return the number of bytes written and error if any
func (s *SSEConnection) Write(bytes []byte) (int, error) {
	write, err := s.writer.Write(bytes)
	if err != nil {
		return 0, err
	}
	s.flusher.Flush()
	return write, nil
}

// Close for user to close the connection
// If you want to close the connection, you should call this function(make sure your bowser and server support with http 2.0 or higher)
func (s *SSEConnection) Close() error {
	if !s.isClosed {
		s.waitChannel <- Close
		s.isClosed = true
	}
	return nil
}

func NewSSEConnection(w http.ResponseWriter, r RequestX.Request) (*SSEConnection, chan string, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return nil, nil, http.ErrNotSupported
	}
	c := make(chan string)
	return &SSEConnection{
		Request:     r,
		waitChannel: c,
		writer:      w,
		flusher:     flusher,
	}, c, nil

}
