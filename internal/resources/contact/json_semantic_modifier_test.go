package contact

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestJSONSemanticEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{
			name: "identical strings",
			a:    `{"a":1}`,
			b:    `{"a":1}`,
			want: true,
		},
		{
			name: "key order does not matter",
			a:    `{"a":1,"b":2}`,
			b:    `{"b":2,"a":1}`,
			want: true,
		},
		{
			name: "whitespace does not matter",
			a:    `{"a": 1,  "b":  2}`,
			b:    `{"a":1,"b":2}`,
			want: true,
		},
		{
			name: "nested objects compared deeply",
			a:    `{"x":{"y":[1,2,{"z":true}]}}`,
			b:    `{"x":{"y":[1,2,{"z":true}]}}`,
			want: true,
		},
		{
			name: "different values are not equal",
			a:    `{"a":1}`,
			b:    `{"a":2}`,
			want: false,
		},
		{
			name: "array order matters",
			a:    `[1,2,3]`,
			b:    `[3,2,1]`,
			want: false,
		},
		{
			name: "missing key is not equal",
			a:    `{"a":1,"b":2}`,
			b:    `{"a":1}`,
			want: false,
		},
		{
			name: "numeric forms both decode to float64",
			a:    `{"a":1}`,
			b:    `{"a":1.0}`,
			want: true,
		},
		{
			name: "number is not the string of that number",
			a:    `{"a":1}`,
			b:    `{"a":"1"}`,
			want: false,
		},
		{
			name: "null is not absent",
			a:    `{"a":null}`,
			b:    `{}`,
			want: false,
		},
		// When either side fails to parse, the function must fall back to a
		// plain string comparison instead of reporting a false match.
		{
			name: "both invalid but identical falls back to string equality",
			a:    "not json",
			b:    "not json",
			want: true,
		},
		{
			name: "both invalid and different",
			a:    "not json",
			b:    "also not json",
			want: false,
		},
		{
			name: "one side invalid is never equal",
			a:    `{"a":1}`,
			b:    "not json",
			want: false,
		},
		{
			name: "empty strings are equal via fallback",
			a:    "",
			b:    "",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := jsonSemanticEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("jsonSemanticEqual(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// jsonSemanticEqual must be symmetric, otherwise the plan modifier would
// behave differently depending on which side Terraform happens to pass.
func TestJSONSemanticEqualIsSymmetric(t *testing.T) {
	t.Parallel()

	pairs := [][2]string{
		{`{"a":1,"b":2}`, `{"b":2,"a":1}`},
		{`{"a":1}`, `{"a":2}`},
		{`{"a":1}`, "not json"},
		{"not json", "not json"},
	}

	for _, p := range pairs {
		ab := jsonSemanticEqual(p[0], p[1])
		ba := jsonSemanticEqual(p[1], p[0])
		if ab != ba {
			t.Errorf("asymmetric for (%q, %q): a==b is %v but b==a is %v", p[0], p[1], ab, ba)
		}
	}
}

func TestPlanModifyString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		state      types.String
		plan       types.String
		wantPlan   types.String
		wantReason string
	}{
		{
			name:       "semantically equal collapses plan onto state",
			state:      types.StringValue(`{"a":1,"b":2}`),
			plan:       types.StringValue(`{"b":2,"a":1}`),
			wantPlan:   types.StringValue(`{"a":1,"b":2}`),
			wantReason: "reordered keys must not produce a diff",
		},
		{
			name:       "genuinely different value is left alone",
			state:      types.StringValue(`{"a":1}`),
			plan:       types.StringValue(`{"a":2}`),
			wantPlan:   types.StringValue(`{"a":2}`),
			wantReason: "a real change must still show up as a diff",
		},
		{
			name:       "identical values are left alone",
			state:      types.StringValue(`{"a":1}`),
			plan:       types.StringValue(`{"a":1}`),
			wantPlan:   types.StringValue(`{"a":1}`),
			wantReason: "no change, nothing to suppress",
		},
		{
			name:       "null state (resource creation) is left alone",
			state:      types.StringNull(),
			plan:       types.StringValue(`{"a":1}`),
			wantPlan:   types.StringValue(`{"a":1}`),
			wantReason: "there is no prior value to compare against",
		},
		{
			name:       "unknown state is left alone",
			state:      types.StringUnknown(),
			plan:       types.StringValue(`{"a":1}`),
			wantPlan:   types.StringValue(`{"a":1}`),
			wantReason: "unknown state cannot be compared",
		},
		{
			name:       "null plan is left alone",
			state:      types.StringValue(`{"a":1}`),
			plan:       types.StringNull(),
			wantPlan:   types.StringNull(),
			wantReason: "removing the value is a real change",
		},
		{
			name:       "unknown plan is left alone",
			state:      types.StringValue(`{"a":1}`),
			plan:       types.StringUnknown(),
			wantPlan:   types.StringUnknown(),
			wantReason: "unknown plan cannot be compared",
		},
		{
			name:       "invalid JSON on both sides with different text stays different",
			state:      types.StringValue("not json"),
			plan:       types.StringValue("also not json"),
			wantPlan:   types.StringValue("also not json"),
			wantReason: "non-JSON values fall back to string comparison",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := JSONSemanticEqual()

			req := planmodifier.StringRequest{
				StateValue: tt.state,
				PlanValue:  tt.plan,
			}
			resp := &planmodifier.StringResponse{PlanValue: tt.plan}

			m.PlanModifyString(context.Background(), req, resp)

			if !resp.PlanValue.Equal(tt.wantPlan) {
				t.Errorf("PlanValue = %v, want %v (%s)", resp.PlanValue, tt.wantPlan, tt.wantReason)
			}
			if resp.Diagnostics.HasError() {
				t.Errorf("unexpected diagnostics: %+v", resp.Diagnostics)
			}
		})
	}
}

func TestJSONSemanticEqualModifierDescriptions(t *testing.T) {
	t.Parallel()

	m := JSONSemanticEqual()
	ctx := context.Background()

	if m.Description(ctx) == "" {
		t.Error("Description() must not be empty")
	}
	if m.MarkdownDescription(ctx) == "" {
		t.Error("MarkdownDescription() must not be empty")
	}
}
