package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stripe/stripe-go/v81/client"
)

// Ensure StripeProvider satisfies various provider interfaces.
var _ provider.Provider = &StripeProvider{}
var _ provider.ProviderWithFunctions = &StripeProvider{}

// StripeProvider defines the provider implementation.
type StripeProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// StripeProviderModel describes the provider data model.
type StripeProviderModel struct {
	APIKey types.String `tfsdk:"api_key"`
}

func (p *StripeProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "stripe"
	resp.Version = p.version
}

func (p *StripeProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "The Stripe API key. Can also be sourced from the `STRIPE_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *StripeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config StripeProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown Stripe API key",
			"The Stripe API key must be set.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("STRIPE_API_KEY")

	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Stripe API key",
			"The Stripe API key must be set.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Example client configuration for data sources and resources
	stripeAPI := client.New(apiKey, nil)
	resp.DataSourceData = stripeAPI
	resp.ResourceData = stripeAPI
}

func (p *StripeProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCouponResource,
		NewPriceResource,
		NewProductResource,
		NewWebhookEndpointResource,
	}
}

func (p *StripeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *StripeProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &StripeProvider{
			version: version,
		}
	}
}
