package handlers

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/flash1nho/GophProfile/internal/services"
)

func newHandler() *AvatarHandler {
	return &AvatarHandler{
		svc: &services.AvatarService{},
		log: zap.NewNop(),
	}
}

func addParam(r *http.Request, key, val string) *http.Request {
	rc := chi.NewRouteContext()
	rc.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rc))
}

func TestUpload_NoUser(t *testing.T) {
	h := newHandler()

	req := httptest.NewRequest("POST", "/", nil)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != 400 {
		t.Fatalf("expected 400")
	}
}

func TestUpload_NoFile(t *testing.T) {
	h := newHandler()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", "1")

	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != 400 {
		t.Fatalf("expected 400")
	}
}

func TestUpload_InvalidMime(t *testing.T) {
	h := newHandler()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("file", "test.txt")
	_, _ = part.Write([]byte("not an image"))

	writer.Close()

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", "1")

	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != 400 {
		t.Fatalf("expected 400")
	}
}

func TestGet_InvalidSize(t *testing.T) {
	h := newHandler()

	req := httptest.NewRequest("GET", "/?size=bad", nil)
	req = addParam(req, "id", "1")

	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != 400 {
		t.Fatalf("expected 400")
	}
}

func TestGet_InvalidFormat(t *testing.T) {
	h := newHandler()

	req := httptest.NewRequest("GET", "/?format=bad", nil)
	req = addParam(req, "id", "1")

	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != 400 {
		t.Fatalf("expected 400")
	}
}

func TestDelete_NoHeader(t *testing.T) {
	h := newHandler()

	req := httptest.NewRequest("DELETE", "/", nil)
	req = addParam(req, "id", "1")

	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != 400 {
		t.Fatalf("expected 400")
	}
}

func TestDeleteByUser_Forbidden(t *testing.T) {
	h := newHandler()

	req := httptest.NewRequest("DELETE", "/", nil)
	req = addParam(req, "user_id", "1")
	req.Header.Set("X-User-ID", "2")

	rec := httptest.NewRecorder()

	h.DeleteByUser(rec, req)

	if rec.Code != 403 {
		t.Fatalf("expected 403")
	}
}

func TestHealth_NoPanic(t *testing.T) {
	h := newHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	defer func() {
		_ = recover()
	}()

	h.Health(rec, req)
}

func TestGetBaseURL(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "localhost"

	if getBaseURL(req) != "http://localhost" {
		t.Fatal("bad url")
	}

	req.Header.Set("X-Forwarded-Proto", "https")

	if getBaseURL(req) != "https://localhost" {
		t.Fatal("bad https url")
	}
}
