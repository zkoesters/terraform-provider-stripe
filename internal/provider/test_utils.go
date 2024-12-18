package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"testing"
)

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
