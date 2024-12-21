package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"stripe": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	if apiKey := os.Getenv("STRIPE_API_KEY"); apiKey == "" {
		t.Fatal("STRIPE_API_KEY must be set for acceptance tests")
	}
}

func testListValue(t *testing.T, elemType attr.Type, vals interface{}) types.List {
	lv, diags := types.ListValueFrom(context.Background(), elemType, vals)
	if diags.HasError() {
		t.Fatalf("failed to construct list value: %s", diags)
	}
	return lv
}

func testSetValue(t *testing.T, elemType attr.Type, vals interface{}) types.Set {
	lv, diags := types.SetValueFrom(context.Background(), elemType, vals)
	if diags.HasError() {
		t.Fatalf("failed to construct list value: %s", diags)
	}
	return lv
}

func testMapValue(t *testing.T, elemType attr.Type, vals map[string]interface{}) types.Map {
	mv, diags := types.MapValueFrom(context.Background(), elemType, vals)
	if diags.HasError() {
		t.Fatalf("failed to construct map value: %s", diags)
	}
	return mv
}
