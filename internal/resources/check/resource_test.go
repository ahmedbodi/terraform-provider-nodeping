package check

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nodeping/terraform-provider-nodeping/internal/client"
)

type credentials struct {
	Username types.String
	Password types.String
}

func TestCredentialsNodePingDoesNotReturnAreKept(t *testing.T) {
	model := CheckResourceModel{
		Username: types.StringValue("admin"),
		Password: types.StringValue("s3cret"),
	}

	(&CheckResource{}).mapCheckToModel(context.Background(), httpCheck(""), &model)

	assertCredentials(t, credentials{types.StringValue("admin"), types.StringValue("s3cret")}, model)
}

func TestAUsernameReturnedByNodePingReplacesTheKnownOne(t *testing.T) {
	model := CheckResourceModel{Username: types.StringValue("admin")}

	(&CheckResource{}).mapCheckToModel(context.Background(), httpCheck("changed-in-nodeping"), &model)

	assertCredentials(t, credentials{types.StringValue("changed-in-nodeping"), types.StringNull()}, model)
}

func httpCheck(username string) *client.Check {
	return &client.Check{
		ID:         "201401131142Z6ERP-ABCDEFGH",
		Type:       "HTTP",
		Parameters: client.CheckParameters{Target: "https://example.com/health", Username: username},
	}
}

func assertCredentials(t *testing.T, want credentials, model CheckResourceModel) {
	t.Helper()
	if got := (credentials{model.Username, model.Password}); got != want {
		t.Errorf("credentials = %+v, want %+v", got, want)
	}
}
