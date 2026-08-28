package check

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

// ValidateImplementation catches schema mistakes that would otherwise only
// surface at runtime for a user: attributes that are neither Required,
// Optional nor Computed, invalid nested block nesting, reserved names, and so
// on.
func TestCheckSchemaIsValid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}

	NewCheckResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned diagnostics: %+v", resp.Diagnostics)
	}

	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("schema validation failed: %+v", diags)
	}
}

func TestCheckSchemaCoreAttributes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}
	NewCheckResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	// `id` is assigned by NodePing and must never be user-settable, otherwise
	// Terraform would try to send it on create.
	id, ok := attrs["id"]
	if !ok {
		t.Fatal("schema is missing the id attribute")
	}
	if !id.IsComputed() {
		t.Error("id must be Computed")
	}
	if id.IsRequired() {
		t.Error("id must not be Required")
	}

	// A check without a type or a target cannot be created.
	for _, name := range []string{"type", "target", "label"} {
		if _, ok := attrs[name]; !ok {
			t.Errorf("schema is missing the %q attribute", name)
		}
	}

	// AUDIO check arguments. Per the NodePing API these are
	// "verifyvolume - optional boolean" and "volumemin - optional integer
	// (acceptable range -90 to 0)".
	for _, name := range []string{"verifyvolume", "volumemin"} {
		attr, ok := attrs[name]
		if !ok {
			t.Errorf("schema is missing the %q attribute", name)
			continue
		}
		if !attr.IsOptional() {
			t.Errorf("%q must be Optional, it only applies to AUDIO checks", name)
		}
		if attr.IsRequired() {
			t.Errorf("%q must not be Required", name)
		}
	}

	// AUDIO has to be an accepted check type, otherwise the documented
	// example config would be rejected by the type validator.
	audio := false
	for _, ct := range ValidCheckTypes {
		if ct == "AUDIO" {
			audio = true
			break
		}
	}
	if !audio {
		t.Error("AUDIO is missing from ValidCheckTypes, so verifyvolume/volumemin are unreachable")
	}

	// Check-type specific arguments must stay Optional: they only apply to a
	// subset of the 30+ NodePing check types.
	for _, name := range []string{"port", "snmpv", "snmpcom", "contentstring", "statuscode"} {
		attr, ok := attrs[name]
		if !ok {
			t.Errorf("schema is missing the %q attribute", name)
			continue
		}
		if attr.IsRequired() {
			t.Errorf("%q must not be Required, it only applies to some check types", name)
		}
	}
}

// Every attribute has to be usable: an attribute that is neither required,
// optional nor computed is dead weight the user can never set or read.
func TestCheckSchemaAttributesAreUsable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}
	NewCheckResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	for name, attr := range resp.Schema.Attributes {
		if !attr.IsRequired() && !attr.IsOptional() && !attr.IsComputed() {
			t.Errorf("attribute %q is neither Required, Optional nor Computed", name)
		}
	}
}

// Descriptions end up in the generated registry documentation, so a missing
// one is a user-facing defect.
func TestCheckSchemaAttributesHaveDescriptions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}
	NewCheckResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	for name, attr := range resp.Schema.Attributes {
		if attr.GetDescription() == "" && attr.GetMarkdownDescription() == "" {
			t.Errorf("attribute %q has no description", name)
		}
	}
}
