package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stripe/stripe-go/v81"
)

func convertListToStringPtrs(tflist types.List) []*string {
	if tflist.IsUnknown() || tflist.IsNull() {
		return nil
	}

	var strings []*string
	for _, element := range tflist.Elements() {
		if element.IsNull() {
			strings = append(strings, nil)
		} else {
			if str, ok := element.(types.String); ok {
				s := str.ValueString()
				strings = append(strings, &s)
			}
		}
	}
	return strings
}

func convertSetToStringPtrs(set types.Set) []*string {
	if set.IsUnknown() || set.IsNull() {
		return nil
	}

	var strings []*string
	for _, element := range set.Elements() {
		if element.IsNull() {
			strings = append(strings, nil)
		} else {
			if str, ok := element.(types.String); ok {
				s := str.ValueString()
				strings = append(strings, &s)
			}
		}
	}
	return strings
}

func Float64NullIfEmpty(input float64) types.Float64 {
	if input == 0 {
		return types.Float64Null()
	}
	return types.Float64Value(input)
}

func Int64NullIfEmpty(input int64) types.Int64 {
	if input == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(input)
}

func StringNullIfEmpty(input string) types.String {
	if input == "" {
		return types.StringNull()
	}
	return types.StringValue(input)
}

func ListValueNullIfEmpty(input types.List, elementType attr.Type) types.List {
	if input.IsNull() || len(input.Elements()) == 0 {
		return types.ListNull(elementType)
	}
	return input
}

func MapValueNullIfEmpty(input types.Map, elementType attr.Type) types.Map {
	if input.IsNull() || len(input.Elements()) == 0 {
		return types.MapNull(elementType)
	}
	return input
}

func EmptyStringIfNull(s basetypes.StringValue) *string {
	if s.IsNull() {
		return stripe.String("")
	}
	return s.ValueStringPointer()
}
