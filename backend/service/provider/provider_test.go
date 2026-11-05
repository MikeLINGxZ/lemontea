package provider

import (
	"context"
	"fmt"
	"testing"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"
	pkgProvider "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider/provider_dto"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/storage"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestProviderService creates an isolated provider service backed by an in-memory SQLite database.
func newTestProviderService(t *testing.T) *Provider {
	t.Helper()
	t.Setenv("LEMONTEA_DATA_DIR", t.TempDir())

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	stor, err := storage.NewStorageFromDB(db)
	if err != nil {
		t.Fatal(err)
	}
	return NewProvider(stor)
}

// TestCreateProviderPersistsRecord verifies that CreateProvider stores a provider retrievable via ListProviders.
func TestCreateProviderPersistsRecord(t *testing.T) {
	svc := newTestProviderService(t)
	ctx := context.Background()

	_, err := svc.CreateProvider(ctx, provider_dto.CreateProviderInput{
		ProviderName: "DeepSeek",
		ProviderType: pkgProvider.Deepseek,
		BaseUrl:      "",
		ApiKey:       "sk-test",
		Enable:       true,
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	out, err := svc.ListProviders(ctx, provider_dto.ListProvidersInput{})
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	if len(out.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(out.Providers))
	}
	if out.Providers[0].Provider.Name != "DeepSeek" {
		t.Fatalf("expected name DeepSeek, got %q", out.Providers[0].Provider.Name)
	}
}

// TestListProvidersReturnsEmpty verifies ListProviders returns an empty list when no providers exist.
func TestListProvidersReturnsEmpty(t *testing.T) {
	svc := newTestProviderService(t)
	ctx := context.Background()

	out, err := svc.ListProviders(ctx, provider_dto.ListProvidersInput{})
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	if len(out.Providers) != 0 {
		t.Fatalf("expected 0 providers, got %d", len(out.Providers))
	}
}

// TestDeleteProviderRemovesRecord verifies that DeleteProvider removes the record from ListProviders.
func TestDeleteProviderRemovesRecord(t *testing.T) {
	svc := newTestProviderService(t)
	ctx := context.Background()

	_, err := svc.CreateProvider(ctx, provider_dto.CreateProviderInput{
		ProviderName: "ToDelete",
		ProviderType: pkgProvider.Deepseek,
		Enable:       true,
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.ListProviders(ctx, provider_dto.ListProvidersInput{})
	if err != nil {
		t.Fatal(err)
	}
	id := out.Providers[0].Provider.ID

	_, err = svc.DeleteProvider(ctx, provider_dto.DeleteProviderInput{ProviderId: int64(id)})
	if err != nil {
		t.Fatalf("DeleteProvider: %v", err)
	}

	out, err = svc.ListProviders(ctx, provider_dto.ListProvidersInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Providers) != 0 {
		t.Fatalf("expected 0 providers after delete, got %d", len(out.Providers))
	}
}

// TestEditProviderUpdatesFields verifies that EditProvider persists updated provider fields.
func TestEditProviderUpdatesFields(t *testing.T) {
	svc := newTestProviderService(t)
	ctx := context.Background()

	_, err := svc.CreateProvider(ctx, provider_dto.CreateProviderInput{
		ProviderName: "Original",
		ProviderType: pkgProvider.Deepseek,
		BaseUrl:      "http://original.test",
		ApiKey:       "sk-old",
		Enable:       true,
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.ListProviders(ctx, provider_dto.ListProvidersInput{})
	if err != nil {
		t.Fatal(err)
	}
	id := out.Providers[0].Provider.ID

	_, err = svc.EditProvider(ctx, provider_dto.EditProviderInput{
		ProviderId:   int64(id),
		ProviderName: "Updated",
		BaseUrl:      "http://updated.test",
		ApiKey:       "sk-new",
		Enable:       false,
	})
	if err != nil {
		t.Fatalf("EditProvider: %v", err)
	}

	out, err = svc.ListProviders(ctx, provider_dto.ListProvidersInput{})
	if err != nil {
		t.Fatal(err)
	}
	got := out.Providers[0].Provider
	if got.Name != "Updated" {
		t.Errorf("expected name Updated, got %q", got.Name)
	}
	if got.BaseURL != "http://updated.test" {
		t.Errorf("expected base_url http://updated.test, got %q", got.BaseURL)
	}
	if got.ApiKey != "sk-new" {
		t.Errorf("expected api_key sk-new, got %q", got.ApiKey)
	}
	if got.Enabled {
		t.Error("expected enabled=false, got true")
	}
}

func TestCreateProviderMarksFirstProviderAndModelDefault(t *testing.T) {
	svc := newTestProviderService(t)
	ctx := context.Background()

	_, err := svc.CreateProvider(ctx, provider_dto.CreateProviderInput{
		ProviderName: "First",
		ProviderType: pkgProvider.Deepseek,
		Enable:       true,
		Models: []view_model.Model{
			{Model: "deepseek-chat", Enable: true},
		},
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	out, err := svc.ProviderAndModelList(ctx, provider_dto.ProviderAndModelListInput{})
	if err != nil {
		t.Fatalf("ProviderAndModelList: %v", err)
	}
	if len(out.ProviderModels) != 1 {
		t.Fatalf("expected 1 provider model wrapper, got %d", len(out.ProviderModels))
	}
	if !out.ProviderModels[0].Provider.IsDefault {
		t.Fatal("expected first provider to be marked default")
	}
	if len(out.ProviderModels[0].Models) != 1 || !out.ProviderModels[0].Models[0].IsDefault {
		t.Fatalf("expected first model to be marked default, got %+v", out.ProviderModels[0].Models)
	}
}
