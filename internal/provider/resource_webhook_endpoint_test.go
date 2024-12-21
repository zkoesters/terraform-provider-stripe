package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	testAccWebhookEndpointResourceConfigCreate = `
resource "stripe_webhook_endpoint" "test" {
  api_version = "2024-09-30.acacia"
  description = "test_create"
  enabled_events = [
    "customer.created"
  ]
  url = "https://example.com/test"
  metadata = {
	foo = "bar"
  }
}
`
	testAccWebhookEndpointResourceConfigUpdate = `
resource "stripe_webhook_endpoint" "test" {
  api_version = "2024-09-30.acacia"
  description = "test_update"
  enabled_events = [
    "customer.created",
    "customer.updated"
  ]
  url = "https://example.com/test_updated"
  metadata = {
	foo = "bar"
    bar = "foo"
  }
}
`
	testAccWebhookEndpointResourceConfigReplace = `
resource "stripe_webhook_endpoint" "test" {
  description = "test_replace"
  enabled_events = [
    "customer.updated"
  ]
  url = "https://example.com/test"
  metadata = {
	foo = "bar"
  }
}
`
)

func TestAccWebhookEndpointResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccWebhookEndpointResourceConfigCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "api_version", "2024-09-30.acacia"),
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "description", "test_create"),
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "url", "https://example.com/test"),
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "enabled_events.#", "1"),
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "metadata.%", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "stripe_webhook_endpoint.test",
				ImportState:       true,
				ImportStateVerify: false,
			},
			// Update and Read testing
			{
				Config: testAccWebhookEndpointResourceConfigUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "api_version", "2024-09-30.acacia"),
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "description", "test_update"),
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "url", "https://example.com/test_updated"),
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "enabled_events.#", "2"),
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "metadata.%", "2"),
				),
			},
			// Replace and Read testing
			{
				Config: testAccWebhookEndpointResourceConfigReplace,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("stripe_webhook_endpoint.test", "api_version"),
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "description", "test_replace"),
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "url", "https://example.com/test"),
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "enabled_events.#", "1"),
					resource.TestCheckResourceAttr("stripe_webhook_endpoint.test", "metadata.%", "1"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestBuildCreateParamsWebhookEndpointResource(t *testing.T) {
	tests := []struct {
		name      string
		plan      WebhookEndpointResourceModel
		expectErr bool
		expected  stripe.WebhookEndpointParams
	}{
		{
			name: "all values provided",
			plan: WebhookEndpointResourceModel{
				EnabledEvents: testSetValue(t, types.StringType, []attr.Value{types.StringValue("event1"), types.StringValue("event2")}),
				URL:           types.StringValue("https://example.com"),
				Description:   types.StringValue("Test description"),
				Metadata:      testMapValue(t, types.StringType, map[string]interface{}{"key": types.StringValue("value")}),
				APIVersion:    types.StringValue("2024-09-30"),
			},
			expectErr: false,
			expected: stripe.WebhookEndpointParams{
				EnabledEvents: stripe.StringSlice([]string{"event1", "event2"}),
				URL:           stripe.String("https://example.com"),
				Description:   stripe.String("Test description"),
				Metadata: map[string]string{
					"key": "value",
				},
				APIVersion: stripe.String("2024-09-30"),
			},
		},
		{
			name: "missing optional values",
			plan: WebhookEndpointResourceModel{
				URL: types.StringValue("https://example.com"),
			},
			expectErr: false,
			expected: stripe.WebhookEndpointParams{
				URL: stripe.String("https://example.com"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &WebhookEndpointResource{}
			params := r.buildCreateParams(tt.plan)
			require.Equal(t, tt.expected.EnabledEvents, params.EnabledEvents, "EnabledEvents should match")
			require.Equal(t, tt.expected.URL, params.URL, "URL should match")
			require.Equal(t, tt.expected.Description, params.Description, "Description should match")
			require.Equal(t, tt.expected.Metadata, params.Metadata, "Metadata should match")
			require.Equal(t, tt.expected.APIVersion, params.APIVersion, "APIVersion should match")
		})
	}
}

func TestBuildUpdateParamsWebhookEndpointResource(t *testing.T) {
	tests := []struct {
		name     string
		state    WebhookEndpointResourceModel
		plan     WebhookEndpointResourceModel
		expected stripe.WebhookEndpointParams
	}{
		{
			name: "update description",
			state: WebhookEndpointResourceModel{
				Description: types.StringValue("Old description"),
			},
			plan: WebhookEndpointResourceModel{
				Description: types.StringValue("New description"),
			},
			expected: stripe.WebhookEndpointParams{
				Description: stripe.String("New description"),
			},
		},
		{
			name: "remove description",
			state: WebhookEndpointResourceModel{
				Description: types.StringValue("Description"),
			},
			plan: WebhookEndpointResourceModel{
				Description: types.StringNull(),
			},
			expected: stripe.WebhookEndpointParams{
				Description: stripe.String(""),
			},
		},
		{
			name: "update enabled events",
			state: WebhookEndpointResourceModel{
				EnabledEvents: testSetValue(t, types.StringType, []attr.Value{types.StringValue("event1")}),
			},
			plan: WebhookEndpointResourceModel{
				EnabledEvents: testSetValue(t, types.StringType, []attr.Value{types.StringValue("event1"), types.StringValue("event2")}),
			},
			expected: stripe.WebhookEndpointParams{
				EnabledEvents: stripe.StringSlice([]string{"event1", "event2"}),
			},
		},
		{
			name: "update metadata",
			state: WebhookEndpointResourceModel{
				Metadata: testMapValue(t, types.StringType, map[string]interface{}{"key": types.StringValue("old_value")}),
			},
			plan: WebhookEndpointResourceModel{
				Metadata: testMapValue(t, types.StringType, map[string]interface{}{"key": types.StringValue("new_value")}),
			},
			expected: stripe.WebhookEndpointParams{
				Metadata: map[string]string{
					"key": "new_value",
				},
			},
		},
		{
			name: "remove metadata",
			state: WebhookEndpointResourceModel{
				Metadata: testMapValue(t, types.StringType, map[string]interface{}{"key": types.StringValue("value")}),
			},
			plan: WebhookEndpointResourceModel{
				Metadata: types.MapNull(types.StringType),
			},
			expected: stripe.WebhookEndpointParams{
				Metadata: map[string]string{
					"key": "",
				},
			},
		},
		{
			name: "update URL",
			state: WebhookEndpointResourceModel{
				URL: types.StringValue("https://old-url.com"),
			},
			plan: WebhookEndpointResourceModel{
				URL: types.StringValue("https://new-url.com"),
			},
			expected: stripe.WebhookEndpointParams{
				URL: stripe.String("https://new-url.com"),
			},
		},
		{
			name: "update status to disabled",
			state: WebhookEndpointResourceModel{
				Disabled: types.BoolValue(false),
			},
			plan: WebhookEndpointResourceModel{
				Disabled: types.BoolValue(true),
			},
			expected: stripe.WebhookEndpointParams{
				Disabled: stripe.Bool(true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &WebhookEndpointResource{}
			params := r.buildUpdateParams(tt.state, tt.plan)

			require.Equal(t, tt.expected.Description, params.Description, "Description should match")
			require.Equal(t, tt.expected.EnabledEvents, params.EnabledEvents, "EnabledEvents should match")
			require.Equal(t, tt.expected.Metadata, params.Metadata, "Metadata should match")
			require.Equal(t, tt.expected.URL, params.URL, "URL should match")
			require.Equal(t, tt.expected.Disabled, params.Disabled, "Disabled should match")
		})
	}
}

func TestPopulateModelWebhookEndpointResource(t *testing.T) {
	tests := []struct {
		name   string
		model  WebhookEndpointResourceModel
		input  stripe.WebhookEndpoint
		expect WebhookEndpointResourceModel
	}{
		{
			name:  "all fields populated",
			model: WebhookEndpointResourceModel{},
			input: stripe.WebhookEndpoint{
				APIVersion:    "2024-09-30",
				Application:   "app_id",
				Description:   "Test description",
				EnabledEvents: []string{"event1", "event2"},
				Metadata:      map[string]string{"key": "value"},
				Status:        "enabled",
				URL:           "https://example.com",
			},
			expect: WebhookEndpointResourceModel{
				APIVersion:    types.StringValue("2024-09-30"),
				Application:   types.StringValue("app_id"),
				Description:   types.StringValue("Test description"),
				Disabled:      types.BoolValue(false),
				EnabledEvents: testSetValue(t, types.StringType, []attr.Value{types.StringValue("event1"), types.StringValue("event2")}),
				Metadata:      testMapValue(t, types.StringType, map[string]interface{}{"key": types.StringValue("value")}),
				URL:           types.StringValue("https://example.com"),
			},
		},
		{
			name:  "empty metadata",
			model: WebhookEndpointResourceModel{},
			input: stripe.WebhookEndpoint{
				APIVersion:    "2024-09-30",
				Application:   "app_id",
				Description:   "Test description",
				EnabledEvents: []string{"event1", "event2"},
				Metadata:      map[string]string{},
				Status:        "enabled",
				URL:           "https://example.com",
			},
			expect: WebhookEndpointResourceModel{
				APIVersion:    types.StringValue("2024-09-30"),
				Application:   types.StringValue("app_id"),
				Description:   types.StringValue("Test description"),
				Disabled:      types.BoolValue(false),
				EnabledEvents: testSetValue(t, types.StringType, []attr.Value{types.StringValue("event1"), types.StringValue("event2")}),
				Metadata:      types.MapNull(types.StringType),
				URL:           types.StringValue("https://example.com"),
			},
		},
		{
			name:  "empty enabled events",
			model: WebhookEndpointResourceModel{},
			input: stripe.WebhookEndpoint{
				APIVersion:    "2024-09-30",
				Application:   "app_id",
				Description:   "Test description",
				EnabledEvents: []string{},
				Metadata:      map[string]string{"key": "value"},
				Status:        "enabled",
				URL:           "https://example.com",
			},
			expect: WebhookEndpointResourceModel{
				APIVersion:    types.StringValue("2024-09-30"),
				Application:   types.StringValue("app_id"),
				Description:   types.StringValue("Test description"),
				Disabled:      types.BoolValue(false),
				EnabledEvents: testSetValue(t, types.StringType, []attr.Value{}),
				Metadata:      testMapValue(t, types.StringType, map[string]interface{}{"key": types.StringValue("value")}),
				URL:           types.StringValue("https://example.com"),
			},
		},
		{
			name:  "missing optional fields",
			model: WebhookEndpointResourceModel{},
			input: stripe.WebhookEndpoint{
				APIVersion:    "",
				Application:   "",
				Description:   "",
				EnabledEvents: []string{},
				Metadata:      map[string]string{},
				Status:        "enabled",
				URL:           "https://example.com",
			},
			expect: WebhookEndpointResourceModel{
				APIVersion:    types.StringNull(),
				Application:   types.StringNull(),
				Description:   types.StringNull(),
				Disabled:      types.BoolValue(false),
				EnabledEvents: testSetValue(t, types.StringType, []attr.Value{}),
				Metadata:      types.MapNull(types.StringType),
				URL:           types.StringValue("https://example.com"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &WebhookEndpointResource{}
			respDiag := diag.Diagnostics{}
			ctx := context.Background()
			r.populateModel(ctx, &tt.model, &tt.input, respDiag)

			require.Equal(t, tt.expect.APIVersion, tt.model.APIVersion, "APIVersion should match")
			require.Equal(t, tt.expect.Application, tt.model.Application, "Application should match")
			require.Equal(t, tt.expect.Description, tt.model.Description, "Description should match")
			require.Equal(t, tt.expect.Disabled, tt.model.Disabled, "Status should match")
			require.Equal(t, tt.expect.EnabledEvents, tt.model.EnabledEvents, "EnabledEvents should match")
			require.Equal(t, tt.expect.Metadata, tt.model.Metadata, "Metadata should match")
			require.Equal(t, tt.expect.URL, tt.model.URL, "URL should match")
		})
	}
}
