package RouteDisPatch

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoute_AddHandler(t *testing.T) {
	pathMap := getPathMap("/m/{name}/1", "/a/b/1/c/d")
	if got := pathMap["name"]; len(got) != 1 || got[0] != "b" {
		t.Fatalf("unexpected path map: %#v", pathMap)
	}
}

func TestServeHTTPRecoversHandlerPanic(t *testing.T) {
	handler := InitHandler()
	handler.Routes.AddHttpHandler("/panic", http.MethodGet, func(http.ResponseWriter, *Request) {
		panic(errors.New("boom"))
	})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	var response errorStruct
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if response.Code != http.StatusInternalServerError || response.Msg != "boom" {
		t.Fatalf("unexpected error response: %+v", response)
	}
}

func TestHeaderParamHandlerAcceptsPointerAndEmptyBody(t *testing.T) {
	type params struct {
		TraceID string `quickParam:"X-Trace-ID"`
	}

	handler := InitHandler()
	var got *params
	handler.Routes.AddHeaderParamHandler("/headers", http.MethodGet, &params{}, func(_ http.ResponseWriter, r *Request) {
		got = r.Param.(*params)
	})
	req := httptest.NewRequest(http.MethodGet, "/headers", nil)
	req.Header.Set("X-Trace-ID", "trace-123")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got == nil || got.TraceID != "trace-123" {
		t.Fatalf("unexpected parameters: %+v", got)
	}
}
