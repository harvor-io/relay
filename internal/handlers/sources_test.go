package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/handlers"
	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/services"
)

// fakeSourceService is a programmable handlers.SourceService.
type fakeSourceService struct {
	getFn    func(context.Context, uuid.UUID) (*models.Source, error)
	listFn   func(context.Context) ([]models.Source, error)
	createFn func(context.Context, services.CreateSourceInput) (*models.Source, error)
	deleteFn func(context.Context, uuid.UUID) error
}

func (f fakeSourceService) Get(ctx context.Context, id uuid.UUID) (*models.Source, error) {
	return f.getFn(ctx, id)
}

func (f fakeSourceService) List(ctx context.Context) ([]models.Source, error) {
	return f.listFn(ctx)
}

func (f fakeSourceService) Create(ctx context.Context, in services.CreateSourceInput) (*models.Source, error) {
	return f.createFn(ctx, in)
}

func (f fakeSourceService) Delete(ctx context.Context, id uuid.UUID) error {
	return f.deleteFn(ctx, id)
}

func newSourcesRouter(svc services.SourceService) *chi.Mux {
	r := chi.NewRouter()
	handlers.NewSourceHandler(svc).RegisterRoutes(r)
	return r
}

func do(t *testing.T, r http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, reader)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func sampleSource() *models.Source {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	desc := "from the shop"
	return &models.Source{
		ID:          uuid.Must(uuid.FromString("018f9c9e-0000-7000-8000-000000000001")),
		Name:        "orders",
		Slug:        "orders",
		Description: &desc,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestListSources(t *testing.T) {
	src := sampleSource()
	r := newSourcesRouter(fakeSourceService{
		listFn: func(context.Context) ([]models.Source, error) {
			return []models.Source{*src}, nil
		},
	})

	rec := do(t, r, http.MethodGet, "/sources", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var got []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0]["id"] != src.ID.String() || got[0]["name"] != "orders" {
		t.Errorf("unexpected resource: %v", got[0])
	}
	if _, ok := got[0]["created_at"]; !ok {
		t.Error("created_at missing from resource")
	}
}

func TestListSourcesEmptyIsArray(t *testing.T) {
	r := newSourcesRouter(fakeSourceService{
		listFn: func(context.Context) ([]models.Source, error) { return nil, nil },
	})

	rec := do(t, r, http.MethodGet, "/sources", "")
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("body = %q, want []", rec.Body.String())
	}
}

func TestGetSource(t *testing.T) {
	src := sampleSource()
	r := newSourcesRouter(fakeSourceService{
		getFn: func(_ context.Context, id uuid.UUID) (*models.Source, error) {
			if id != src.ID {
				t.Errorf("id = %s, want %s", id, src.ID)
			}
			return src, nil
		},
	})

	rec := do(t, r, http.MethodGet, "/sources/"+src.ID.String(), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["name"] != "orders" {
		t.Errorf("name = %v, want orders", got["name"])
	}
}

func TestGetSourceInvalidID(t *testing.T) {
	r := newSourcesRouter(fakeSourceService{})
	rec := do(t, r, http.MethodGet, "/sources/not-a-uuid", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestGetSourceNotFound(t *testing.T) {
	r := newSourcesRouter(fakeSourceService{
		getFn: func(context.Context, uuid.UUID) (*models.Source, error) {
			return nil, services.ErrSourceNotFound
		},
	})
	rec := do(t, r, http.MethodGet, "/sources/"+uuid.Must(uuid.NewV7()).String(), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var got map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&got)
	if got["error"] == "" {
		t.Error("error message missing")
	}
}

func TestCreateSource(t *testing.T) {
	src := sampleSource()
	var gotInput services.CreateSourceInput
	r := newSourcesRouter(fakeSourceService{
		createFn: func(_ context.Context, in services.CreateSourceInput) (*models.Source, error) {
			gotInput = in
			return src, nil
		},
	})

	rec := do(t, r, http.MethodPost, "/sources", `{"name":"orders","description":"from the shop"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if gotInput.Name != "orders" || gotInput.Description == nil || *gotInput.Description != "from the shop" {
		t.Errorf("input = %+v", gotInput)
	}
	var got map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["id"] != src.ID.String() || got["slug"] != src.Slug {
		t.Errorf("resource = %v", got)
	}
}

func TestCreateSourcePassesSlug(t *testing.T) {
	src := sampleSource()
	var gotInput services.CreateSourceInput
	r := newSourcesRouter(fakeSourceService{
		createFn: func(_ context.Context, in services.CreateSourceInput) (*models.Source, error) {
			gotInput = in
			return src, nil
		},
	})

	do(t, r, http.MethodPost, "/sources", `{"name":"Orders","slug":"shop-orders"}`)
	if gotInput.Slug != "shop-orders" {
		t.Errorf("Slug = %q, want %q", gotInput.Slug, "shop-orders")
	}
}

func TestCreateSourceSlugTaken(t *testing.T) {
	r := newSourcesRouter(fakeSourceService{
		createFn: func(context.Context, services.CreateSourceInput) (*models.Source, error) {
			return nil, services.ErrSourceSlugTaken
		},
	})
	rec := do(t, r, http.MethodPost, "/sources", `{"name":"Orders","slug":"orders"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

func TestCreateSourceInvalidJSON(t *testing.T) {
	r := newSourcesRouter(fakeSourceService{})
	rec := do(t, r, http.MethodPost, "/sources", `{"name":`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateSourceValidationError(t *testing.T) {
	r := newSourcesRouter(fakeSourceService{
		createFn: func(context.Context, services.CreateSourceInput) (*models.Source, error) {
			return nil, services.ErrSourceNameRequired
		},
	})
	rec := do(t, r, http.MethodPost, "/sources", `{"name":""}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	var got map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&got)
	if got["error"] != services.ErrSourceNameRequired.Error() {
		t.Errorf("error = %q, want %q", got["error"], services.ErrSourceNameRequired.Error())
	}
}

func TestDeleteSource(t *testing.T) {
	called := false
	r := newSourcesRouter(fakeSourceService{
		deleteFn: func(context.Context, uuid.UUID) error {
			called = true
			return nil
		},
	})
	rec := do(t, r, http.MethodDelete, "/sources/"+uuid.Must(uuid.NewV7()).String(), "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if !called {
		t.Error("service Delete was not called")
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestDeleteSourceNotFound(t *testing.T) {
	r := newSourcesRouter(fakeSourceService{
		deleteFn: func(context.Context, uuid.UUID) error {
			return services.ErrSourceNotFound
		},
	})
	rec := do(t, r, http.MethodDelete, "/sources/"+uuid.Must(uuid.NewV7()).String(), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
