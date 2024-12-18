package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go/v81"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	testAccProductResourceConfigCreate string = `
resource "stripe_product" "test" {
  name = "test"
  metadata = {
	test = "test"
  }
}
`
	testAccProductResourceConfigUpdate string = `
resource "stripe_product" "test" {
  name = "test_updated"
  metadata = {
	test = "test"
  }
}
`
)

func TestAccProductResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProductResourceConfigCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stripe_product.test", "name", "test"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "stripe_product.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config:  testAccProductResourceConfigUpdate,
				Destroy: false,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stripe_product.test", "name", "test_updated"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestPopulateModelProductResource(t *testing.T) {
	tests := []struct {
		name       string
		product    *stripe.Product
		expected   ProductResourceModel
		expectDiag bool
	}{
		{
			name: "All fields filled",
			product: &stripe.Product{
				Active:       true,
				DefaultPrice: &stripe.Price{ID: "price_123"},
				Description:  "A product",
				Images:       []string{"image1", "image2"},
				MarketingFeatures: []*stripe.ProductMarketingFeature{
					{Name: "Feature 1"},
				},
				Metadata: map[string]string{
					"foo": "bar",
				},
				Name: "Product 1",
				PackageDimensions: &stripe.ProductPackageDimensions{
					Height: 1.5,
					Length: 2.0,
					Weight: 0.5,
					Width:  1.0,
				},
				Shippable:           true,
				StatementDescriptor: "Descriptor",
				TaxCode:             &stripe.TaxCode{ID: "tax_123"},
				UnitLabel:           "unit",
				URL:                 "http://example.com",
			},
			expected: ProductResourceModel{
				Active:              types.BoolValue(true),
				DefaultPrice:        types.StringValue("price_123"),
				Description:         types.StringValue("A product"),
				Images:              testListValue(t, types.StringType, []string{"image1", "image2"}),
				MarketingFeatures:   testListValue(t, types.StringType, []string{"Feature 1"}),
				Metadata:            testMapValue(t, types.StringType, map[string]interface{}{"foo": "bar"}),
				Name:                types.StringValue("Product 1"),
				PackageDimensions:   buildPackageDimensionsModel(t, 1.5, 2.0, 0.5, 1.0),
				Shippable:           types.BoolValue(true),
				StatementDescriptor: types.StringValue("Descriptor"),
				TaxCode:             types.StringValue("tax_123"),
				UnitLabel:           types.StringValue("unit"),
				URL:                 types.StringValue("http://example.com"),
			},
			expectDiag: false,
		},
		{
			name: "Empty fields",
			product: &stripe.Product{
				Active:            false,
				DefaultPrice:      nil,
				Description:       "",
				Images:            []string{},
				MarketingFeatures: []*stripe.ProductMarketingFeature{},
				Metadata:          map[string]string{},
				Name:              "",
				PackageDimensions: &stripe.ProductPackageDimensions{
					Height: 0,
					Length: 0,
					Weight: 0,
					Width:  0,
				},
				Shippable:           false,
				StatementDescriptor: "",
				TaxCode:             nil,
				UnitLabel:           "",
				URL:                 "",
			},
			expected: ProductResourceModel{
				Active:              types.BoolValue(false),
				DefaultPrice:        types.StringNull(),
				Description:         types.StringNull(),
				Images:              types.ListNull(types.StringType),
				MarketingFeatures:   types.ListNull(types.StringType),
				Metadata:            testMapValue(t, types.StringType, nil),
				Name:                types.StringValue(""),
				PackageDimensions:   types.ObjectNull(ProductPackageDimensionsResourceModel{}.Types()),
				Shippable:           types.BoolValue(false),
				StatementDescriptor: types.StringNull(),
				TaxCode:             types.StringNull(),
				UnitLabel:           types.StringNull(),
				URL:                 types.StringNull(),
			},
			expectDiag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var model ProductResourceModel
			var diags diag.Diagnostics

			r := &ProductResource{}
			r.populateModel(context.Background(), &model, tt.product, diags)

			assert.Equal(t, tt.expected, model)
			if tt.expectDiag {
				assert.True(t, diags.HasError())
			} else {
				assert.False(t, diags.HasError())
			}
		})
	}
}

func TestBuildCreateParamsProductResource(t *testing.T) {
	tests := []struct {
		name     string
		plan     ProductResourceModel
		expected *stripe.ProductParams
	}{
		{
			name: "All fields set",
			plan: ProductResourceModel{
				Id:                  types.StringValue("prod_123"),
				Active:              types.BoolValue(true),
				DefaultPrice:        types.StringValue("price_123"),
				Description:         types.StringValue("A product"),
				Images:              testListValue(t, types.StringType, []string{"image1", "image2"}),
				MarketingFeatures:   testListValue(t, types.StringType, []string{"Feature 1"}),
				Metadata:            testMapValue(t, types.StringType, map[string]interface{}{"foo": "bar"}),
				Name:                types.StringValue("Product 1"),
				PackageDimensions:   buildPackageDimensionsModel(t, 1.5, 2.0, 0.5, 1.0),
				Shippable:           types.BoolValue(true),
				StatementDescriptor: types.StringValue("Descriptor"),
				TaxCode:             types.StringValue("tax_123"),
				UnitLabel:           types.StringValue("unit"),
				URL:                 types.StringValue("http://example.com"),
			},
			expected: &stripe.ProductParams{
				ID:                  stripe.String("prod_123"),
				Active:              stripe.Bool(true),
				DefaultPrice:        stripe.String("price_123"),
				Description:         stripe.String("A product"),
				Images:              []*string{stripe.String("image1"), stripe.String("image2")},
				MarketingFeatures:   []*stripe.ProductMarketingFeatureParams{{Name: stripe.String("Feature 1")}},
				Metadata:            map[string]string{"foo": "bar"},
				Name:                stripe.String("Product 1"),
				PackageDimensions:   &stripe.ProductPackageDimensionsParams{Height: stripe.Float64(1.5), Length: stripe.Float64(2.0), Weight: stripe.Float64(0.5), Width: stripe.Float64(1.0)},
				Shippable:           stripe.Bool(true),
				StatementDescriptor: stripe.String("Descriptor"),
				TaxCode:             stripe.String("tax_123"),
				UnitLabel:           stripe.String("unit"),
				URL:                 stripe.String("http://example.com"),
			},
		},
		{
			name: "No optional fields set",
			plan: ProductResourceModel{
				Name: types.StringValue("Product 2"),
			},
			expected: &stripe.ProductParams{
				Name: stripe.String("Product 2"),
			},
		},
		{
			name: "Empty fields",
			plan: ProductResourceModel{
				Id:                  types.StringValue(""),
				Active:              types.BoolValue(false),
				DefaultPrice:        types.StringNull(),
				Description:         types.StringNull(),
				Images:              types.ListNull(types.StringType),
				MarketingFeatures:   types.ListNull(types.StringType),
				Metadata:            testMapValue(t, types.StringType, nil),
				Name:                types.StringValue(""),
				PackageDimensions:   types.ObjectNull(ProductPackageDimensionsResourceModel{}.Types()),
				Shippable:           types.BoolValue(false),
				StatementDescriptor: types.StringNull(),
				TaxCode:             types.StringNull(),
				UnitLabel:           types.StringNull(),
				URL:                 types.StringNull(),
			},
			expected: &stripe.ProductParams{
				Active:    stripe.Bool(false),
				ID:        stripe.String(""),
				Name:      stripe.String(""),
				Shippable: stripe.Bool(false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ProductResource{}
			respDiag := diag.Diagnostics{}
			params := r.buildCreateParams(context.Background(), tt.plan, respDiag)
			assert.Equal(t, tt.expected, params)
		})
	}
}

func TestBuildUpdateParamsProductResource(t *testing.T) {
	tests := []struct {
		name     string
		state    ProductResourceModel
		plan     ProductResourceModel
		expected *stripe.ProductParams
	}{
		{
			name: "All fields updated",
			state: ProductResourceModel{
				Active:              types.BoolValue(false),
				DefaultPrice:        types.StringValue("price_123"),
				Description:         types.StringValue("An old product"),
				Images:              testListValue(t, types.StringType, []string{"old_image1", "old_image2"}),
				MarketingFeatures:   testListValue(t, types.StringType, []string{"Old Feature"}),
				Metadata:            testMapValue(t, types.StringType, map[string]interface{}{"key": "value"}),
				Name:                types.StringValue("Old Product"),
				PackageDimensions:   buildPackageDimensionsModel(t, 0.5, 1.0, 0.2, 0.8),
				Shippable:           types.BoolValue(false),
				StatementDescriptor: types.StringValue("Old Descriptor"),
				TaxCode:             types.StringValue("old_tax_123"),
				UnitLabel:           types.StringValue("old_unit"),
				URL:                 types.StringValue("http://oldexample.com"),
			},
			plan: ProductResourceModel{
				Active:              types.BoolValue(true),
				DefaultPrice:        types.StringValue("price_456"),
				Description:         types.StringValue("A new product"),
				Images:              testListValue(t, types.StringType, []string{"new_image1", "new_image2"}),
				MarketingFeatures:   testListValue(t, types.StringType, []string{"New Feature"}),
				Metadata:            testMapValue(t, types.StringType, map[string]interface{}{"foo": "bar"}),
				Name:                types.StringValue("New Product"),
				PackageDimensions:   buildPackageDimensionsModel(t, 1.5, 2.0, 0.5, 1.0),
				Shippable:           types.BoolValue(true),
				StatementDescriptor: types.StringValue("New Descriptor"),
				TaxCode:             types.StringValue("new_tax_123"),
				UnitLabel:           types.StringValue("new_unit"),
				URL:                 types.StringValue("http://newexample.com"),
			},
			expected: &stripe.ProductParams{
				Active:              stripe.Bool(true),
				DefaultPrice:        stripe.String("price_456"),
				Description:         stripe.String("A new product"),
				Images:              []*string{stripe.String("new_image1"), stripe.String("new_image2")},
				MarketingFeatures:   []*stripe.ProductMarketingFeatureParams{{Name: stripe.String("New Feature")}},
				Metadata:            map[string]string{"key": "", "foo": "bar"},
				Name:                stripe.String("New Product"),
				PackageDimensions:   &stripe.ProductPackageDimensionsParams{Height: stripe.Float64(1.5), Length: stripe.Float64(2.0), Weight: stripe.Float64(0.5), Width: stripe.Float64(1.0)},
				Shippable:           stripe.Bool(true),
				StatementDescriptor: stripe.String("New Descriptor"),
				TaxCode:             stripe.String("new_tax_123"),
				UnitLabel:           stripe.String("new_unit"),
				URL:                 stripe.String("http://newexample.com"),
			},
		},
		{
			name: "No fields updated",
			state: ProductResourceModel{
				Name: types.StringValue("Product 2"),
			},
			plan: ProductResourceModel{
				Name: types.StringValue("Product 2"),
			},
			expected: &stripe.ProductParams{
				MarketingFeatures: []*stripe.ProductMarketingFeatureParams{},
			},
		},
		{
			name: "Only Active field updated",
			state: ProductResourceModel{
				Active: types.BoolValue(false),
			},
			plan: ProductResourceModel{
				Active: types.BoolValue(true),
			},
			expected: &stripe.ProductParams{
				Active:            stripe.Bool(true),
				MarketingFeatures: []*stripe.ProductMarketingFeatureParams{},
			},
		},
		{
			name: "Only Images updated",
			state: ProductResourceModel{
				Images: testListValue(t, types.StringType, []string{"old_image1"}),
			},
			plan: ProductResourceModel{
				Images: testListValue(t, types.StringType, []string{"new_image1", "new_image2"}),
			},
			expected: &stripe.ProductParams{
				Images:            []*string{stripe.String("new_image1"), stripe.String("new_image2")},
				MarketingFeatures: []*stripe.ProductMarketingFeatureParams{},
			},
		},
		{
			name: "Only Metadata updated",
			state: ProductResourceModel{
				Metadata: testMapValue(t, types.StringType, map[string]interface{}{"key1": "value1"}),
			},
			plan: ProductResourceModel{
				Metadata: testMapValue(t, types.StringType, map[string]interface{}{"key2": "value2"}),
			},
			expected: &stripe.ProductParams{
				MarketingFeatures: []*stripe.ProductMarketingFeatureParams{},
				Metadata: map[string]string{
					"key1": "",
					"key2": "value2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ProductResource{}
			respDiag := diag.Diagnostics{}
			params := r.buildUpdateParams(context.Background(), tt.state, tt.plan, respDiag)
			assert.Equal(t, tt.expected, params)
		})
	}
}

func buildPackageDimensionsModel(t *testing.T, height, length, weight, width float64) types.Object {
	p, diags := types.ObjectValueFrom(
		context.Background(),
		map[string]attr.Type{
			"height": types.Float64Type,
			"length": types.Float64Type,
			"weight": types.Float64Type,
			"width":  types.Float64Type,
		},
		&ProductPackageDimensionsResourceModel{
			Height: types.Float64Value(height),
			Length: types.Float64Value(length),
			Weight: types.Float64Value(weight),
			Width:  types.Float64Value(width),
		},
	)
	if diags.HasError() {
		t.Fatalf("failed to construct package dimensions object value: %s", diags)
	}
	return p
}
