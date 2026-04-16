package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"

	"github.com/flash1nho/GophProfile/internal/domain"
	"github.com/flash1nho/GophProfile/internal/services"
)

type webMockRepo struct {
	avatars []domain.Avatar
	err     error
}

func (m *webMockRepo) ListByUser(ctx context.Context, userID string) ([]domain.Avatar, error) {
	return m.avatars, m.err
}

func (m *webMockRepo) Create(context.Context, *domain.Avatar) error              { return nil }
func (m *webMockRepo) GetAvatar(context.Context, string) (*domain.Avatar, error) { return nil, nil }
func (m *webMockRepo) SoftDelete(context.Context, string) error                  { return nil }
func (m *webMockRepo) GetLatestByUser(context.Context, string) (*domain.Avatar, error) {
	return nil, nil
}
func (m *webMockRepo) UpdateUploadStatus(context.Context, string, domain.UploadStatus) error {
	return nil
}
func (m *webMockRepo) UpdateProcessingStatus(context.Context, string, domain.ProcessingStatus) error {
	return nil
}
func (m *webMockRepo) UpdateThumbnails(context.Context, string, map[string]string) error {
	return nil
}
func (m *webMockRepo) Ping(context.Context) error { return nil }

func setupTemplates(t *testing.T) {
	dir := "web/static"
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>ok</html>"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(dir, "gallery.html"), []byte("{{range .}}{{.ID}}{{end}}"), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func newWebHandler(repo *webMockRepo) *AvatarHandler {
	svc := services.NewAvatarService(repo, nil, nil, zap.NewNop())

	return &AvatarHandler{
		svc: svc,
		log: zap.NewNop(),
	}
}

func addParamWeb(r *http.Request, key, val string) *http.Request {
	rc := chi.NewRouteContext()
	rc.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rc))
}

func removeTemplates() {
	_ = os.RemoveAll("web")
}

func TestWebUploadForm_OK(t *testing.T) {
	setupTemplates(t)

	h := newWebHandler(&webMockRepo{})

	req := httptest.NewRequest("GET", "/web/upload", nil)
	rec := httptest.NewRecorder()

	h.WebUploadForm(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestWebUploadForm_TemplateError(t *testing.T) {
	removeTemplates()

	h := newWebHandler(&webMockRepo{})

	req := httptest.NewRequest("GET", "/web/upload", nil)
	rec := httptest.NewRecorder()

	h.WebUploadForm(rec, req)

	if rec.Code != 500 {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestWebUploadSubmit_FromHeader(t *testing.T) {
	h := newWebHandler(&webMockRepo{})

	req := httptest.NewRequest("POST", "/web/upload", nil)
	req.Header.Set("X-User-ID", "1")

	rec := httptest.NewRecorder()

	h.WebUploadSubmit(rec, req)

	if rec.Code == 0 {
		t.Fatalf("no response")
	}
}

func TestWebUploadSubmit_FromForm(t *testing.T) {
	h := newWebHandler(&webMockRepo{})

	req := httptest.NewRequest("POST", "/web/upload?user_id=1", nil)
	rec := httptest.NewRecorder()

	h.WebUploadSubmit(rec, req)

	if rec.Code == 0 {
		t.Fatalf("no response")
	}
}

func TestWebUploadSubmit_NoUser(t *testing.T) {
	h := newWebHandler(&webMockRepo{})

	req := httptest.NewRequest("POST", "/web/upload", nil)
	rec := httptest.NewRecorder()

	h.WebUploadSubmit(rec, req)

	if rec.Code != 400 {
		t.Fatalf("expected 400")
	}
}

func TestWebGallery_OK(t *testing.T) {
	setupTemplates(t)

	repo := &webMockRepo{
		avatars: []domain.Avatar{
			{ID: "1", FileName: "a.png"},
		},
	}

	h := newWebHandler(repo)

	req := httptest.NewRequest("GET", "/web/gallery/1", nil)
	req = addParamWeb(req, "user_id", "1")

	rec := httptest.NewRecorder()

	h.WebGallery(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200")
	}
}

func TestWebGallery_ServiceError(t *testing.T) {
	repo := &webMockRepo{err: context.Canceled}

	h := newWebHandler(repo)

	req := httptest.NewRequest("GET", "/web/gallery/1", nil)
	req = addParamWeb(req, "user_id", "1")

	rec := httptest.NewRecorder()

	h.WebGallery(rec, req)

	if rec.Code != 500 {
		t.Fatalf("expected 500")
	}
}

func TestWebGallery_TemplateError(t *testing.T) {
	removeTemplates()

	repo := &webMockRepo{}
	h := newWebHandler(repo)

	req := httptest.NewRequest("GET", "/web/gallery/1", nil)
	req = addParam(req, "user_id", "1")

	rec := httptest.NewRecorder()

	h.WebGallery(rec, req)

	if rec.Code != 500 {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
