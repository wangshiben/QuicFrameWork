package RouteDisPatch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type injectionParams struct {
	Age      int    `quickLoc:"param"`
	Role     string `quickLoc:"header" quickParam:"X-Role"`
	Name     string
	Fallback string `quickLoc:"param" quickDefault:"fallback"`
}

func TestReflectBackToStructLegacyTags(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/users/alice?age=42", nil)
	req.Header.Set("X-Role", "admin")

	got := reflectBackToStructAsInterface(&injectionParams{}, req, body, "/users/{name}").(*injectionParams)
	if got.Age != 42 || got.Role != "admin" || got.Name != "alice" || got.Fallback != "fallback" {
		t.Fatalf("unexpected injected parameters: %+v", got)
	}
}

func TestReflectBackToStructQuicTag(t *testing.T) {
	type params struct {
		Limit int `quic:"limit,location=param,default=10"`
	}

	req := httptest.NewRequest(http.MethodGet, "/items?limit=25", nil)
	got := reflectBackToStructAsInterface(&params{}, req, body, "/items").(*params)
	if got.Limit != 25 {
		t.Fatalf("Limit = %d, want 25", got.Limit)
	}
}
