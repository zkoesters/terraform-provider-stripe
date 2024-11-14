// Copyright (c) 2024 Zachary Koesters
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestConvertListToStringPtrs(t *testing.T) {
	tests := []struct {
		name string
		list types.List
		want []*string
	}{
		{"null", types.ListNull(types.StringType), nil},
		{"unknown", types.ListUnknown(types.StringType), nil},
		{"empty", types.ListValueMust(types.StringType, []attr.Value{}), nil},
		{"single", types.ListValueMust(types.StringType, []attr.Value{types.StringValue("test")}), []*string{ptr("test")}},
		{"multiple", types.ListValueMust(types.StringType, []attr.Value{types.StringValue("test1"), types.StringValue("test2")}), []*string{ptr("test1"), ptr("test2")}},
		{"with null", types.ListValueMust(types.StringType, []attr.Value{types.StringValue("test1"), types.StringNull()}), []*string{ptr("test1"), nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertListToStringPtrs(tt.list); !equalStringPtrSlices(got, tt.want) {
				t.Errorf("convertListToStringPtrs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFloat64NullIfEmpty(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  types.Float64
	}{
		{"empty", 0, types.Float64Null()},
		{"non-empty", 1.23, types.Float64Value(1.23)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Float64NullIfEmpty(tt.input); !got.Equal(tt.want) {
				t.Errorf("Float64NullIfEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInt64NullIfEmpty(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  types.Int64
	}{
		{"empty", 0, types.Int64Null()},
		{"non-empty", 123, types.Int64Value(123)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Int64NullIfEmpty(tt.input); !got.Equal(tt.want) {
				t.Errorf("Int64NullIfEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringNullIfEmpty(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  types.String
	}{
		{"empty", "", types.StringNull()},
		{"non-empty", "test", types.StringValue("test")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringNullIfEmpty(tt.input); !got.Equal(tt.want) {
				t.Errorf("StringNullIfEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListValueNullIfEmpty(t *testing.T) {
	tests := []struct {
		name        string
		input       types.List
		elementType attr.Type
		want        types.List
	}{
		{"null", types.ListNull(types.StringType), types.StringType, types.ListNull(types.StringType)},
		{"empty", types.ListValueMust(types.StringType, []attr.Value{}), types.StringType, types.ListNull(types.StringType)},
		{"non-empty", types.ListValueMust(types.StringType, []attr.Value{types.StringValue("test")}), types.StringType, types.ListValueMust(types.StringType, []attr.Value{types.StringValue("test")})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ListValueNullIfEmpty(tt.input, tt.elementType); !got.Equal(tt.want) {
				t.Errorf("ListValueNullIfEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapValueNullIfEmpty(t *testing.T) {
	tests := []struct {
		name        string
		input       types.Map
		elementType attr.Type
		want        types.Map
	}{
		{"null", types.MapNull(types.StringType), types.StringType, types.MapNull(types.StringType)},
		{"empty", types.MapValueMust(types.StringType, map[string]attr.Value{}), types.StringType, types.MapNull(types.StringType)},
		{"non-empty", types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}), types.StringType, types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapValueNullIfEmpty(tt.input, tt.elementType); !got.Equal(tt.want) {
				t.Errorf("MapValueNullIfEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func ptr(s string) *string {
	return &s
}

func equalStringPtrSlices(a, b []*string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if (a[i] == nil) != (b[i] == nil) || (a[i] != nil && *a[i] != *b[i]) {
			return false
		}
	}
	return true
}
