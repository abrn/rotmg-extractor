package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func sampleNotification() Notification {
	return Notification{
		Env: "Production", BuildType: "client", Version: "6.11.0.0.0", Hash: "b11b4e834d86",
		NewFiles: 2, DelFiles: 1, AddedLines: 342, RemovedLines: 958,
		ObjAdded: 5, ObjRemoved: 1, ObjChanged: 12,
		GndAdded: 0, GndRemoved: 0, GndChanged: 3,
	}
}

func TestBuildPayload(t *testing.T) {
	p := buildPayload(sampleNotification(), "999")
	content := p["content"].(string)
	if !strings.Contains(content, "Production Client") {
		t.Errorf("content missing env/type: %q", content)
	}
	if !strings.Contains(content, "<@&999>") {
		t.Errorf("content missing role ping: %q", content)
	}

	// No role => no ping.
	p2 := buildPayload(sampleNotification(), "")
	if strings.Contains(p2["content"].(string), "<@&") {
		t.Errorf("unexpected role ping when roleID empty")
	}

	// Embed should be serialisable and contain the version.
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), "6.11.0.0.0") {
		t.Errorf("payload missing version")
	}
}

func TestDiscordNotify(t *testing.T) {
	var gotBody string
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	d := &Discord{WebhookURL: srv.URL, RoleID: "999"}
	if err := d.Notify(context.Background(), sampleNotification()); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if !strings.Contains(gotBody, "b11b4e834d86") {
		t.Errorf("posted body missing hash: %s", gotBody)
	}
}

func TestDiscordNotifyError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	d := &Discord{WebhookURL: srv.URL}
	if err := d.Notify(context.Background(), sampleNotification()); err == nil {
		t.Error("expected error on 400 response")
	}
}
