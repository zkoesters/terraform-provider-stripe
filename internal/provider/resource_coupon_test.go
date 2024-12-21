package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go/v81"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	testAccCouponResourceConfigCreate string = `
resource "stripe_coupon" "test" {
  name = "test"
  currency_options = {
    "usd" = {
      amount_off = 1000
      top_level = true
    }
  }
  duration = "once"
  metadata = {
	test = "test"
  }
}
`
	testAccCouponResourceConfigUpdate string = `
resource "stripe_coupon" "test" {
  name = "test_updated"
  currency_options = {
    "usd" = {
      amount_off = 1000
      top_level = true
    }
  }
  duration = "once"
  metadata = {
	test = "test"
  }
}
`
	testAccCouponResourceConfigReplace string = `
resource "stripe_coupon" "test" {
  name = "test_updated_again"
  currency_options = {
    "usd" = {
      amount_off = 2000
      top_level = true
    }
  }
  duration = "once"
  metadata = {
	test = "test"
  }
}
`
)

func TestAccCouponResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCouponResourceConfigCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stripe_coupon.test", "name", "test"),
					resource.TestCheckResourceAttr("stripe_coupon.test", "duration", "once"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "stripe_coupon.test",
				ImportState:       true,
				ImportStateVerify: false,
			},
			// Update and Read testing
			{
				Config: testAccCouponResourceConfigUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stripe_coupon.test", "name", "test_updated"),
					resource.TestCheckResourceAttr("stripe_coupon.test", "duration", "once"),
				),
			},
			// Replace and Read testing
			{
				Config: testAccCouponResourceConfigReplace,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stripe_coupon.test", "name", "test_updated_again"),
					resource.TestCheckResourceAttr("stripe_coupon.test", "duration", "once"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestPopulateModelCouponResource(t *testing.T) {
	cases := []struct {
		name string
		in   *stripe.Coupon
		want CouponResourceModel
	}{
		{
			name: "Empty coupon options",
			in:   &stripe.Coupon{},
			want: CouponResourceModel{
				AppliesTo: types.ListNull(types.StringType),
				CurrencyOptions: types.MapNull(types.ObjectType{
					AttrTypes: CouponCurrencyOptionsModel{}.Types(),
				}),
				Duration:         types.StringNull(),
				DurationInMonths: types.Int64Null(),
				MaxRedemptions:   types.Int64Null(),
				Metadata:         types.MapNull(types.StringType),
				Name:             types.StringNull(),
				PercentOff:       types.Float64Null(),
				RedeemBy:         types.Int64Null(),
			},
		},
		{
			name: "Full coupon options",
			in: &stripe.Coupon{
				AmountOff: int64(1000),
				AppliesTo: &stripe.CouponAppliesTo{
					Products: []string{"product_1", "product_2"},
				},
				Currency: "usd",
				CurrencyOptions: map[string]*stripe.CouponCurrencyOptions{
					"usd": {
						AmountOff: int64(1000),
					},
				},
				Duration:         stripe.CouponDurationOnce,
				DurationInMonths: int64(6),
				MaxRedemptions:   int64(5),
				Metadata: map[string]string{
					"test": "test_metadata",
				},
				Name:       "test_name",
				PercentOff: float64(25),
				RedeemBy:   int64(1629484800),
			},
			want: CouponResourceModel{
				AppliesTo: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("product_1"),
					types.StringValue("product_2"),
				}),
				CurrencyOptions: types.MapValueMust(
					types.ObjectType{
						AttrTypes: CouponCurrencyOptionsModel{}.Types(),
					},
					map[string]attr.Value{
						"usd": types.ObjectValueMust(CouponCurrencyOptionsModel{}.Types(), map[string]attr.Value{
							"amount_off": types.Int64Value(1000),
							"top_level":  types.BoolValue(true),
						}),
					},
				),
				Duration:         types.StringValue(string(stripe.CouponDurationOnce)),
				DurationInMonths: types.Int64Value(6),
				MaxRedemptions:   types.Int64Value(5),
				Metadata:         types.MapValueMust(types.StringType, map[string]attr.Value{"test": types.StringValue("test_metadata")}),
				Name:             types.StringValue("test_name"),
				PercentOff:       types.Float64Value(25),
				RedeemBy:         types.Int64Value(1629484800),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cr := &CouponResource{}
			var model CouponResourceModel
			diags := diag.Diagnostics{}
			cr.populateModel(context.Background(), &model, tc.in, diags)

			if !assert.ElementsMatch(t, model.AppliesTo.Elements(), tc.want.AppliesTo.Elements()) {
				t.Errorf("unexpected result for AppliesTo: %v", model.AppliesTo.Elements())
			}
			if !assert.Equal(t, model.CurrencyOptions.Elements(), tc.want.CurrencyOptions.Elements()) {
				t.Errorf("unexpected result for CurrencyOptions: %v", model.CurrencyOptions.Elements())
			}
			if !assert.Equal(t, model.Duration, tc.want.Duration) {
				t.Errorf("unexpected result for Duration: %v", model.Duration)
			}
			if !assert.Equal(t, model.DurationInMonths, tc.want.DurationInMonths) {
				t.Errorf("unexpected result for DurationInMonths: %v", model.DurationInMonths)
			}
			if !assert.Equal(t, model.MaxRedemptions, tc.want.MaxRedemptions) {
				t.Errorf("unexpected result for MaxRedemptions: %v", model.MaxRedemptions)
			}
			if !assert.Equal(t, model.Metadata.Elements(), tc.want.Metadata.Elements()) {
				t.Errorf("unexpected result for Metadata: %v", model.Metadata.Elements())
			}
			if !assert.Equal(t, model.Name, tc.want.Name) {
				t.Errorf("unexpected result for Name: %v", model.Name)
			}
			if !assert.Equal(t, model.PercentOff, tc.want.PercentOff) {
				t.Errorf("unexpected result for PercentOff: %v", model.PercentOff)
			}
			if !assert.Equal(t, model.RedeemBy, tc.want.RedeemBy) {
				t.Errorf("unexpected result for RedeemBy: %v", model.RedeemBy)
			}
		})
	}
}

func TestBuildCreateParamsCouponResource(t *testing.T) {
	cases := []struct {
		name string
		data CouponResourceModel
		want *stripe.CouponParams
	}{
		{
			name: "Empty coupon options",
			data: CouponResourceModel{},
			want: &stripe.CouponParams{},
		},
		{
			name: "Full coupon options",
			data: CouponResourceModel{
				AppliesTo: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("product_1"),
					types.StringValue("product_2"),
				}),
				CurrencyOptions: types.MapValueMust(
					types.ObjectType{
						AttrTypes: CouponCurrencyOptionsModel{}.Types(),
					},
					map[string]attr.Value{
						"usd": types.ObjectValueMust(CouponCurrencyOptionsModel{}.Types(), map[string]attr.Value{
							"amount_off": types.Int64Value(1000),
							"top_level":  types.BoolValue(true),
						}),
					},
				),
				Duration:         types.StringValue(string(stripe.CouponDurationOnce)),
				DurationInMonths: types.Int64Value(6),
				MaxRedemptions:   types.Int64Value(5),
				Metadata:         types.MapValueMust(types.StringType, map[string]attr.Value{"test": types.StringValue("test_metadata")}),
				Name:             types.StringValue("test_name"),
				PercentOff:       types.Float64Value(25),
				RedeemBy:         types.Int64Value(1629484800),
			},
			want: &stripe.CouponParams{
				ID:        stripe.String(""),
				AmountOff: stripe.Int64(1000),
				AppliesTo: &stripe.CouponAppliesToParams{
					Products: []*string{
						stripe.String("product_1"),
						stripe.String("product_2"),
					},
				},
				Currency:         stripe.String("usd"),
				CurrencyOptions:  map[string]*stripe.CouponCurrencyOptionsParams{},
				Duration:         stripe.String("once"),
				DurationInMonths: stripe.Int64(6),
				MaxRedemptions:   stripe.Int64(5),
				Metadata: map[string]string{
					"test": "test_metadata",
				},
				Name:       stripe.String("test_name"),
				PercentOff: stripe.Float64(25),
				RedeemBy:   stripe.Int64(1629484800),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cr := &CouponResource{}
			diags := diag.Diagnostics{}
			params := cr.buildCreateParams(context.Background(), tc.data, diags)

			if !assert.Equal(t, tc.want.AmountOff, params.AmountOff) {
				t.Errorf("unexpected result for AmountOff: %v", params.AmountOff)
			}
			if !assert.Equal(t, tc.want.AppliesTo, params.AppliesTo) {
				t.Errorf("unexpected result for AppliesTo: %v", params.AppliesTo)
			}
			if !assert.Equal(t, tc.want.Currency, params.Currency) {
				t.Errorf("unexpected result for Currency: %v", params.Currency)
			}
			if !assert.Equal(t, tc.want.CurrencyOptions, params.CurrencyOptions) {
				t.Errorf("unexpected result for CurrencyOptions: %v", params.CurrencyOptions)
			}
			if !assert.Equal(t, tc.want.Duration, params.Duration) {
				t.Errorf("unexpected result for Duration: %v", params.Duration)
			}
			if !assert.Equal(t, tc.want.DurationInMonths, params.DurationInMonths) {
				t.Errorf("unexpected result for DurationInMonths: %v", params.DurationInMonths)
			}
			if !assert.Equal(t, tc.want.Metadata, params.Metadata) {
				t.Errorf("unexpected result for Metadata: %v", params.Metadata)
			}
			if !assert.Equal(t, tc.want.Name, params.Name) {
				t.Errorf("unexpected result for Name: %v", params.Name)
			}
			if !assert.Equal(t, tc.want.PercentOff, params.PercentOff) {
				t.Errorf("unexpected result for PercentOff: %v", params.PercentOff)
			}
			if !assert.Equal(t, tc.want.RedeemBy, params.RedeemBy) {
				t.Errorf("unexpected result for RedeemBy: %v", params.RedeemBy)
			}
		})
	}
}

func TestBuildUpdateParamsCouponResource(t *testing.T) {
	cases := []struct {
		name  string
		state CouponResourceModel
		plan  CouponResourceModel
		want  *stripe.CouponParams
	}{
		{
			name: "no change",
			state: CouponResourceModel{
				CurrencyOptions: types.MapNull(types.ObjectType{
					AttrTypes: CouponCurrencyOptionsModel{}.Types(),
				}),
				Name:     types.StringValue("test_name"),
				Metadata: types.MapNull(types.StringType),
			},
			plan: CouponResourceModel{
				CurrencyOptions: types.MapNull(types.ObjectType{
					AttrTypes: CouponCurrencyOptionsModel{}.Types(),
				}),
				Name:     types.StringValue("test_name"),
				Metadata: types.MapNull(types.StringType),
			},
			want: &stripe.CouponParams{},
		},
		{
			name: "change name only",
			state: CouponResourceModel{
				CurrencyOptions: types.MapNull(types.ObjectType{
					AttrTypes: CouponCurrencyOptionsModel{}.Types(),
				}),
				Name:     types.StringValue("old_name"),
				Metadata: types.MapNull(types.StringType),
			},
			plan: CouponResourceModel{
				CurrencyOptions: types.MapNull(types.ObjectType{
					AttrTypes: CouponCurrencyOptionsModel{}.Types(),
				}),
				Name:     types.StringValue("new_name"),
				Metadata: types.MapNull(types.StringType),
			},
			want: &stripe.CouponParams{
				Name: stripe.String("new_name"),
			},
		},
		{
			name: "remove name only",
			state: CouponResourceModel{
				CurrencyOptions: types.MapNull(types.ObjectType{
					AttrTypes: CouponCurrencyOptionsModel{}.Types(),
				}),
				Name:     types.StringValue("name"),
				Metadata: types.MapNull(types.StringType),
			},
			plan: CouponResourceModel{
				CurrencyOptions: types.MapNull(types.ObjectType{
					AttrTypes: CouponCurrencyOptionsModel{}.Types(),
				}),
				Name:     types.StringNull(),
				Metadata: types.MapNull(types.StringType),
			},
			want: &stripe.CouponParams{
				Name: stripe.String(""),
			},
		},
		{
			name: "change metadata only",
			state: CouponResourceModel{
				CurrencyOptions: types.MapNull(types.ObjectType{
					AttrTypes: CouponCurrencyOptionsModel{}.Types(),
				}),
				Name:     types.StringValue("test_name"),
				Metadata: types.MapValueMust(types.StringType, map[string]attr.Value{"meta1": types.StringValue("value1")}),
			},
			plan: CouponResourceModel{
				CurrencyOptions: types.MapNull(types.ObjectType{
					AttrTypes: CouponCurrencyOptionsModel{}.Types(),
				}),
				Name:     types.StringValue("test_name"),
				Metadata: types.MapValueMust(types.StringType, map[string]attr.Value{"meta2": types.StringValue("value2")}),
			},
			want: &stripe.CouponParams{
				Metadata: map[string]string{
					"meta1": "",
					"meta2": "value2",
				},
			},
		},
		{
			name: "add currency options only",
			state: CouponResourceModel{
				CurrencyOptions: types.MapValueMust(
					types.ObjectType{
						AttrTypes: CouponCurrencyOptionsModel{}.Types(),
					},
					map[string]attr.Value{
						"usd": types.ObjectValueMust(CouponCurrencyOptionsModel{}.Types(), map[string]attr.Value{
							"amount_off": types.Int64Value(1000),
							"top_level":  types.BoolValue(true),
						}),
					},
				),
				Name: types.StringValue("test_name"),
			},
			plan: CouponResourceModel{
				CurrencyOptions: types.MapValueMust(
					types.ObjectType{
						AttrTypes: CouponCurrencyOptionsModel{}.Types(),
					},
					map[string]attr.Value{
						"usd": types.ObjectValueMust(CouponCurrencyOptionsModel{}.Types(), map[string]attr.Value{
							"amount_off": types.Int64Value(1000),
							"top_level":  types.BoolValue(true),
						}),
						"gbp": types.ObjectValueMust(CouponCurrencyOptionsModel{}.Types(), map[string]attr.Value{
							"amount_off": types.Int64Value(1000),
							"top_level":  types.BoolValue(true),
						}),
					},
				),
				Name: types.StringValue("test_name"),
			},
			want: &stripe.CouponParams{
				AmountOff: nil,
				Currency:  nil,
				CurrencyOptions: map[string]*stripe.CouponCurrencyOptionsParams{
					"gbp": {
						AmountOff: stripe.Int64(1000),
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cr := &CouponResource{}
			diags := diag.Diagnostics{}
			params := cr.buildUpdateParams(context.Background(), tc.state, tc.plan, diags)

			if !assert.Equal(t, tc.want, params) {
				t.Errorf("unexpected result for %s: %v", tc.name, params)
			}
		})
	}
}
