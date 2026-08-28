package checks

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestDataSourceMetadata(t *testing.T) {
	t.Parallel()

	resp := &fwdatasource.MetadataResponse{}
	NewChecksDataSource().Metadata(context.Background(), fwdatasource.MetadataRequest{ProviderTypeName: "nodeping"}, resp)

	if want := "nodeping_checks"; resp.TypeName != want {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, want)
	}
}

// ValidateImplementation catches schema defects that would otherwise only
// appear when a user actually runs terraform against this data source.
func TestDataSourceSchemaIsValid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	resp := &fwdatasource.SchemaResponse{}

	NewChecksDataSource().Schema(ctx, fwdatasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned diagnostics: %+v", resp.Diagnostics)
	}

	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("schema validation failed: %+v", diags)
	}
}

func TestDataSourceSchemaAttributesAreUsable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	resp := &fwdatasource.SchemaResponse{}
	NewChecksDataSource().Schema(ctx, fwdatasource.SchemaRequest{}, resp)

	if len(resp.Schema.Attributes) == 0 && len(resp.Schema.Blocks) == 0 {
		t.Fatal("schema exposes neither attributes nor blocks")
	}

	for name, attr := range resp.Schema.Attributes {
		if !attr.IsRequired() && !attr.IsOptional() && !attr.IsComputed() {
			t.Errorf("attribute %q is neither Required, Optional nor Computed", name)
		}
	}
}
