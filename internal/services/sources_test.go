package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/repositories"
	"github.com/harvor-io/relay/internal/services"
)

// fakeSourceRepo is an in-memory repositories.SourceRepository for tests.
type fakeSourceRepo struct {
	items   map[uuid.UUID]models.Source
	listErr error
}

func newFakeSourceRepo() *fakeSourceRepo {
	return &fakeSourceRepo{items: map[uuid.UUID]models.Source{}}
}

func (f *fakeSourceRepo) Create(_ context.Context, source *models.Source) error {
	for _, existing := range f.items {
		if existing.Slug == source.Slug {
			return repositories.ErrConflict
		}
	}
	if source.ID.IsNil() {
		source.ID = uuid.Must(uuid.NewV7())
	}
	now := time.Now().UTC()
	if source.CreatedAt.IsZero() {
		source.CreatedAt = now
	}
	source.UpdatedAt = now
	f.items[source.ID] = *source
	return nil
}

func (f *fakeSourceRepo) Get(_ context.Context, id uuid.UUID) (*models.Source, error) {
	source, ok := f.items[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return &source, nil
}

func (f *fakeSourceRepo) GetBySlug(_ context.Context, slug string) (*models.Source, error) {
	for _, source := range f.items {
		if source.Slug == slug {
			return &source, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeSourceRepo) List(_ context.Context) ([]models.Source, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]models.Source, 0, len(f.items))
	for _, s := range f.items {
		out = append(out, s)
	}
	return out, nil
}

func (f *fakeSourceRepo) Update(_ context.Context, source *models.Source) error {
	if _, ok := f.items[source.ID]; !ok {
		return repositories.ErrNotFound
	}
	f.items[source.ID] = *source
	return nil
}

func (f *fakeSourceRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.items[id]; !ok {
		return repositories.ErrNotFound
	}
	delete(f.items, id)
	return nil
}

// fakeEnvelopeRepo is an in-memory repositories.EnvelopeRepository for tests.
type fakeEnvelopeRepo struct {
	items map[uuid.UUID]models.Envelope
}

func newFakeEnvelopeRepo() *fakeEnvelopeRepo {
	return &fakeEnvelopeRepo{items: map[uuid.UUID]models.Envelope{}}
}

func (f *fakeEnvelopeRepo) Create(_ context.Context, envelope *models.Envelope) error {
	if envelope.ID.IsNil() {
		envelope.ID = uuid.Must(uuid.NewV7())
	}
	if envelope.CreatedAt.IsZero() {
		envelope.CreatedAt = time.Now().UTC()
	}
	f.items[envelope.ID] = *envelope
	return nil
}

func (f *fakeEnvelopeRepo) Get(_ context.Context, id uuid.UUID) (*models.Envelope, error) {
	envelope, ok := f.items[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return &envelope, nil
}

func (f *fakeEnvelopeRepo) List(_ context.Context) ([]models.Envelope, error) {
	out := make([]models.Envelope, 0, len(f.items))
	for _, e := range f.items {
		out = append(out, e)
	}
	return out, nil
}

func TestSourceServiceCreate(t *testing.T) {
	t.Run("trims fields and assigns identity", func(t *testing.T) {
		svc := services.NewSourceService(newFakeSourceRepo(), newFakeEnvelopeRepo())

		source, err := svc.Create(context.Background(), services.CreateSourceInput{
			Name:        "  orders  ",
			Description: new("  from the shop  "),
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if source.Name != "orders" {
			t.Errorf("Name = %q, want %q", source.Name, "orders")
		}
		if source.Slug != "orders" {
			t.Errorf("Slug = %q, want %q", source.Slug, "orders")
		}
		if source.Description == nil || *source.Description != "from the shop" {
			t.Errorf("Description = %v, want %q", source.Description, "from the shop")
		}
		if source.ID.IsNil() {
			t.Error("ID was not assigned")
		}
		if source.CreatedAt.IsZero() || source.UpdatedAt.IsZero() {
			t.Error("timestamps were not assigned")
		}
	})

	t.Run("blank description becomes nil", func(t *testing.T) {
		svc := services.NewSourceService(newFakeSourceRepo(), newFakeEnvelopeRepo())

		source, err := svc.Create(context.Background(), services.CreateSourceInput{
			Name:        "orders",
			Description: new("   "),
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if source.Description != nil {
			t.Errorf("Description = %v, want nil", *source.Description)
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		svc := services.NewSourceService(newFakeSourceRepo(), newFakeEnvelopeRepo())

		_, err := svc.Create(context.Background(), services.CreateSourceInput{Name: "   "})
		if !errors.Is(err, services.ErrSourceNameRequired) {
			t.Fatalf("err = %v, want ErrSourceNameRequired", err)
		}
	})

	t.Run("rejects over-long name", func(t *testing.T) {
		svc := services.NewSourceService(newFakeSourceRepo(), newFakeEnvelopeRepo())

		_, err := svc.Create(context.Background(), services.CreateSourceInput{
			Name: strings.Repeat("a", 256),
		})
		if !errors.Is(err, services.ErrSourceNameTooLong) {
			t.Fatalf("err = %v, want ErrSourceNameTooLong", err)
		}
	})

	t.Run("rejects over-long description", func(t *testing.T) {
		svc := services.NewSourceService(newFakeSourceRepo(), newFakeEnvelopeRepo())

		_, err := svc.Create(context.Background(), services.CreateSourceInput{
			Name:        "orders",
			Description: new(strings.Repeat("a", 1025)),
		})
		if !errors.Is(err, services.ErrSourceDescriptionTooLong) {
			t.Fatalf("err = %v, want ErrSourceDescriptionTooLong", err)
		}
	})
}

func TestSourceServiceCreateSlug(t *testing.T) {
	t.Run("derives a url-safe slug from the name", func(t *testing.T) {
		svc := services.NewSourceService(newFakeSourceRepo(), newFakeEnvelopeRepo())

		source, err := svc.Create(context.Background(), services.CreateSourceInput{
			Name: "Order Events (US)",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if source.Slug != "order-events-us" {
			t.Errorf("Slug = %q, want %q", source.Slug, "order-events-us")
		}
	})

	t.Run("suffixes a derived slug that collides", func(t *testing.T) {
		svc := services.NewSourceService(newFakeSourceRepo(), newFakeEnvelopeRepo())

		first, err := svc.Create(context.Background(), services.CreateSourceInput{Name: "orders"})
		if err != nil {
			t.Fatalf("Create first: %v", err)
		}
		second, err := svc.Create(context.Background(), services.CreateSourceInput{Name: "Orders"})
		if err != nil {
			t.Fatalf("Create second: %v", err)
		}
		if first.Slug != "orders" || second.Slug != "orders-2" {
			t.Errorf("slugs = %q, %q; want orders, orders-2", first.Slug, second.Slug)
		}
	})

	t.Run("keeps an explicit valid slug", func(t *testing.T) {
		svc := services.NewSourceService(newFakeSourceRepo(), newFakeEnvelopeRepo())

		source, err := svc.Create(context.Background(), services.CreateSourceInput{
			Name: "Orders",
			Slug: "shop-orders",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if source.Slug != "shop-orders" {
			t.Errorf("Slug = %q, want %q", source.Slug, "shop-orders")
		}
	})

	t.Run("rejects a non-url-safe explicit slug", func(t *testing.T) {
		svc := services.NewSourceService(newFakeSourceRepo(), newFakeEnvelopeRepo())

		_, err := svc.Create(context.Background(), services.CreateSourceInput{
			Name: "Orders",
			Slug: "Shop Orders!",
		})
		if !errors.Is(err, services.ErrSourceSlugInvalid) {
			t.Fatalf("err = %v, want ErrSourceSlugInvalid", err)
		}
	})

	t.Run("rejects an explicit slug already in use", func(t *testing.T) {
		repo := newFakeSourceRepo()
		svc := services.NewSourceService(repo, newFakeEnvelopeRepo())

		if _, err := svc.Create(context.Background(), services.CreateSourceInput{
			Name: "Orders",
			Slug: "orders",
		}); err != nil {
			t.Fatalf("Create first: %v", err)
		}
		_, err := svc.Create(context.Background(), services.CreateSourceInput{
			Name: "More orders",
			Slug: "orders",
		})
		if !errors.Is(err, services.ErrSourceSlugTaken) {
			t.Fatalf("err = %v, want ErrSourceSlugTaken", err)
		}
	})

	t.Run("rejects a name no slug can be derived from", func(t *testing.T) {
		svc := services.NewSourceService(newFakeSourceRepo(), newFakeEnvelopeRepo())

		_, err := svc.Create(context.Background(), services.CreateSourceInput{Name: "!!!"})
		if !errors.Is(err, services.ErrSourceSlugUnderivable) {
			t.Fatalf("err = %v, want ErrSourceSlugUnderivable", err)
		}
	})
}

func TestSourceServiceGet(t *testing.T) {
	repo := newFakeSourceRepo()
	svc := services.NewSourceService(repo, newFakeEnvelopeRepo())

	created, err := svc.Create(context.Background(), services.CreateSourceInput{Name: "orders"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := svc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %s, want %s", got.ID, created.ID)
	}

	if _, err := svc.Get(context.Background(), uuid.Must(uuid.NewV7())); !errors.Is(err, services.ErrSourceNotFound) {
		t.Fatalf("err = %v, want ErrSourceNotFound", err)
	}
}

func TestSourceServiceDelete(t *testing.T) {
	repo := newFakeSourceRepo()
	svc := services.NewSourceService(repo, newFakeEnvelopeRepo())

	created, err := svc.Create(context.Background(), services.CreateSourceInput{Name: "orders"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := svc.Delete(context.Background(), created.ID); !errors.Is(err, services.ErrSourceNotFound) {
		t.Fatalf("err = %v, want ErrSourceNotFound", err)
	}
}
