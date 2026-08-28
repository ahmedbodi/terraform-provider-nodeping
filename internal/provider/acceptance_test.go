package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/nodeping/terraform-provider-nodeping/internal/provider"
	"github.com/nodeping/terraform-provider-nodeping/testutil"
)

// These tests drive a real `terraform plan`/`apply` cycle against an in-process
// mock of the NodePing API. They exercise the parts unit tests cannot reach:
// schema/state round-tripping, plan consistency and the full CRUD lifecycle.
//
// They only run when TF_ACC=1 and need a `terraform` binary on PATH (see
// `make test-acceptance`, which supplies one via Docker).

func protoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"nodeping": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

// providerConfig points the provider at the mock server instead of the real
// NodePing API.
func providerConfig(url string) string {
	return fmt.Sprintf(`
provider "nodeping" {
  api_token = "acc-test-token"
  api_url   = %q
}
`, url)
}

func TestAccContactResource_lifecycle(t *testing.T) {
	mock := testutil.NewMockNodePingServer()
	t.Cleanup(mock.Close)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create
			{
				Config: providerConfig(mock.URL()) + `
resource "nodeping_contact" "test" {
  name = "acc-contact"

  address {
    type    = "email"
    address = "acc@example.com"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nodeping_contact.test", "name", "acc-contact"),
					resource.TestCheckResourceAttrSet("nodeping_contact.test", "id"),
					resource.TestCheckResourceAttr("nodeping_contact.test", "address.#", "1"),
					resource.TestCheckResourceAttr("nodeping_contact.test", "address.0.address", "acc@example.com"),
					resource.TestCheckResourceAttr("nodeping_contact.test", "address.0.type", "email"),
				),
			},
			// Update in place
			{
				Config: providerConfig(mock.URL()) + `
resource "nodeping_contact" "test" {
  name = "acc-contact-renamed"

  address {
    type    = "email"
    address = "acc@example.com"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nodeping_contact.test", "name", "acc-contact-renamed"),
				),
			},
		},
	})
}

func TestAccCheckResource_lifecycle(t *testing.T) {
	mock := testutil.NewMockNodePingServer()
	t.Cleanup(mock.Close)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create
			{
				Config: providerConfig(mock.URL()) + `
resource "nodeping_check" "test" {
  type     = "HTTP"
  target   = "https://example.com"
  label    = "acc-check"
  interval = 5
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nodeping_check.test", "label", "acc-check"),
					resource.TestCheckResourceAttr("nodeping_check.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("nodeping_check.test", "target", "https://example.com"),
					resource.TestCheckResourceAttr("nodeping_check.test", "interval", "5"),
					resource.TestCheckResourceAttrSet("nodeping_check.test", "id"),
				),
			},
			// Update the label in place
			{
				Config: providerConfig(mock.URL()) + `
resource "nodeping_check" "test" {
  type     = "HTTP"
  target   = "https://example.com"
  label    = "acc-check-renamed"
  interval = 5
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nodeping_check.test", "label", "acc-check-renamed"),
				),
			},
		},
	})
}

// A second plan straight after apply must be empty. This is the check that
// catches the "provider produced inconsistent result after apply" class of bug,
// where the API echoes values back in a different shape than the config.
func TestAccCheckResource_planIsEmptyAfterApply(t *testing.T) {
	mock := testutil.NewMockNodePingServer()
	t.Cleanup(mock.Close)

	config := providerConfig(mock.URL()) + `
resource "nodeping_check" "idempotent" {
  type     = "HTTP"
  target   = "https://example.com"
  label    = "acc-idempotent"
  interval = 5
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				Config:   config,
				PlanOnly: true, // fails the test if the plan is not empty
			},
		},
	})
}

func TestAccContactResource_planIsEmptyAfterApply(t *testing.T) {
	mock := testutil.NewMockNodePingServer()
	t.Cleanup(mock.Close)

	config := providerConfig(mock.URL()) + `
resource "nodeping_contact" "idempotent" {
  name = "acc-idempotent"

  address {
    type    = "email"
    address = "idem@example.com"
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// Schema validators must reject bad input at plan time, before any API call
// happens. Terraform never reaches the provider's CRUD code for these.
func TestAccCheckResource_rejectsInvalidValues(t *testing.T) {
	mock := testutil.NewMockNodePingServer()
	t.Cleanup(mock.Close)

	tests := []struct {
		name       string
		body       string
		wantErrRex *regexp.Regexp
	}{
		{
			name: "unknown check type",
			body: `
  type   = "NOT_A_REAL_TYPE"
  target = "https://example.com"
`,
			wantErrRex: regexp.MustCompile(`(?is)type`),
		},
		{
			name: "unsupported http method",
			body: `
  type   = "HTTP"
  target = "https://example.com"
  method = "FETCH"
`,
			wantErrRex: regexp.MustCompile(`(?is)method`),
		},
		{
			name: "warningdays below minimum",
			body: `
  type        = "SSL"
  target      = "https://example.com"
  warningdays = 0
`,
			wantErrRex: regexp.MustCompile(`(?is)warningdays`),
		},
		{
			name: "unsupported snmp version",
			body: `
  type   = "SNMP"
  target = "1.2.3.4"
  snmpv  = "3"
`,
			wantErrRex: regexp.MustCompile(`(?is)snmpv`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: protoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: providerConfig(mock.URL()) + `
resource "nodeping_check" "invalid" {` + tt.body + `}
`,
						ExpectError: tt.wantErrRex,
					},
				},
			})
		})
	}
}

func TestAccContactResource_rejectsInvalidValues(t *testing.T) {
	mock := testutil.NewMockNodePingServer()
	t.Cleanup(mock.Close)

	tests := []struct {
		name       string
		config     string
		wantErrRex *regexp.Regexp
	}{
		{
			name: "unknown custrole",
			config: `
resource "nodeping_contact" "invalid" {
  name     = "bad"
  custrole = "superuser"

  address {
    type    = "email"
    address = "a@example.com"
  }
}
`,
			wantErrRex: regexp.MustCompile(`(?is)custrole`),
		},
		{
			name: "unknown address type",
			config: `
resource "nodeping_contact" "invalid" {
  name = "bad"

  address {
    type    = "carrier-pigeon"
    address = "a@example.com"
  }
}
`,
			wantErrRex: regexp.MustCompile(`(?is)type`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: protoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:      providerConfig(mock.URL()) + tt.config,
						ExpectError: tt.wantErrRex,
					},
				},
			})
		})
	}
}

// Regression test for the unknown-value crash: a check created while the
// provider has no default_tags configured leaves `tags` unknown at plan time.
// Converting that unknown into []string used to abort the apply with
// "Value Conversion Error".
func TestAccCheckResource_appliesWithoutDefaultTags(t *testing.T) {
	mock := testutil.NewMockNodePingServer()
	t.Cleanup(mock.Close)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				// Note: no default_tags on the provider and no tags on the
				// resource. This is the default configuration.
				Config: providerConfig(mock.URL()) + `
resource "nodeping_check" "no_tags" {
  type   = "HTTP"
  target = "https://example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nodeping_check.no_tags", "id"),
				),
			},
		},
	})
}
