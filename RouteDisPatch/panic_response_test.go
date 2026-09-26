package RouteDisPatch

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeHTTPReplacesBufferedResponseAfterPanic(t *testing.T) {
	handler := InitHandler()
	handler.Routes.AddHttpHandler("/panic-after-write", http.MethodGet, func(w http.ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("partial response"))
		panic(errors.New("boom after write"))
	})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic-after-write", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if strings.Contains(recorder.Body.String(), "partial response") {
		t.Fatalf("response contains discarded handler output: %q", recorder.Body.String())
	}
	var response errorStruct
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if response.Code != http.StatusInternalServerError || response.Msg != "boom after write" {
		t.Fatalf("unexpected error response: %+v", response)
	}
}
