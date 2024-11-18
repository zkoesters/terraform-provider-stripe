// Copyright (c) 2024 Zachary Koesters
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/client"
	"github.com/zkoesters/terraform-provider-stripe/internal/provider/planmodifier/customboolplanmodifier"
	"regexp"
)

var _ resource.Resource = &WebhookEndpointResource{}
var _ resource.ResourceWithConfigure = &WebhookEndpointResource{}
var _ resource.ResourceWithImportState = &WebhookEndpointResource{}

func NewWebhookEndpointResource() resource.Resource {
	return &WebhookEndpointResource{}
}

// WebhookEndpointResource defines the resource implementation.
type WebhookEndpointResource struct {
	sc *client.API
}

// WebhookEndpointResourceModel describes the resource data model.
type WebhookEndpointResourceModel struct {
	Id            types.String `tfsdk:"id"`
	APIVersion    types.String `tfsdk:"api_version"`
	Application   types.String `tfsdk:"application"`
	Description   types.String `tfsdk:"description"`
	Disabled      types.Bool   `tfsdk:"disabled"`
	EnabledEvents types.List   `tfsdk:"enabled_events"`
	Metadata      types.Map    `tfsdk:"metadata"`
	Secret        types.String `tfsdk:"secret"`
	URL           types.String `tfsdk:"url"`
}

func (r *WebhookEndpointResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_endpoint"
}

func (r *WebhookEndpointResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A webhook endpoint resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the object",
				Computed:            true,
				Required:            false,
				Optional:            false,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"api_version": schema.StringAttribute{
				MarkdownDescription: "The API version events are rendered as for this webhook endpoint.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"application": schema.StringAttribute{
				MarkdownDescription: "The ID of the associated Connect application.",
				Computed:            true,
				Required:            false,
				Optional:            false,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "An optional description of what the webhook is used for.",
				Optional:            true,
			},
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "Disable the webhook endpoint if set to `true`.",
				Default:             booldefault.StaticBool(false),
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					customboolplanmodifier.DisallowOnCreateOnValue(true),
				},
			},
			"enabled_events": schema.ListAttribute{
				MarkdownDescription: "The list of events to enable for this endpoint. `['*']` indicates that all events are enabled, except those that require explicit selection.",
				ElementType:         types.StringType,
				Required:            true,
				Validators: []validator.List{
					listvalidator.UniqueValues(),
				},
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Set of key-value pairs that you can attach to an object.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators: []validator.Map{
					mapvalidator.SizeAtMost(50),
					mapvalidator.KeysAre(
						stringvalidator.LengthAtMost(40)),
					mapvalidator.ValueStringsAre(
						stringvalidator.LengthAtMost(500)),
				},
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "The endpoint’s secret, used to generate webhook signatures.",
				Computed:            true,
				Sensitive:           true,
				Required:            false,
				Optional:            false,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL of the webhook endpoint.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^https://`),
						"must be a valid HTTPS URL"),
				},
			},
		},
	}
}

func (r *WebhookEndpointResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	sc, ok := req.ProviderData.(*client.API)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.API, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.sc = sc
}

func (r *WebhookEndpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WebhookEndpointResourceModel
	var webhookEndpoint *stripe.WebhookEndpoint
	var err error

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	params := r.buildCreateParams(plan)

	webhookEndpoint, err = r.sc.WebhookEndpoints.New(params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create webhook endpoint, got error: %s", err))
		return
	}

	plan.Id = types.StringValue(webhookEndpoint.ID)
	plan.Secret = types.StringValue(webhookEndpoint.Secret)
	r.populateModel(ctx, &plan, webhookEndpoint, resp.Diagnostics)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookEndpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WebhookEndpointResourceModel
	var webhookEndpoint *stripe.WebhookEndpoint
	var err error

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	webhookEndpoint, err = r.sc.WebhookEndpoints.Get(state.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read webhook endpoint, got error: %s", err))
		return
	}

	r.populateModel(ctx, &state, webhookEndpoint, resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WebhookEndpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan WebhookEndpointResourceModel
	var webhookEndpoint *stripe.WebhookEndpoint
	var err error

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := r.buildUpdateParams(state, plan)

	webhookEndpoint, err = r.sc.WebhookEndpoints.Update(plan.Id.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create webhook endpoint, got error: %s", err))
		return
	}
	r.populateModel(ctx, &plan, webhookEndpoint, resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookEndpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WebhookEndpointResourceModel
	var err error

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err = r.sc.WebhookEndpoints.Del(state.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete webhook endpoint, got error: %s", err))
		return
	}
}

func (r *WebhookEndpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var state WebhookEndpointResourceModel
	var webhookEndpoint *stripe.WebhookEndpoint
	var err error

	webhookEndpoint, err = r.sc.WebhookEndpoints.Get(req.ID, nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import webhook endpoint, got error: %s", err))
		return
	}

	state.Id = types.StringValue(req.ID)
	r.populateModel(ctx, &state, webhookEndpoint, resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WebhookEndpointResource) populateModel(ctx context.Context, model *WebhookEndpointResourceModel, webhookEndpoint *stripe.WebhookEndpoint, respDiag diag.Diagnostics) {
	model.APIVersion = StringNullIfEmpty(webhookEndpoint.APIVersion)
	model.Application = StringNullIfEmpty(webhookEndpoint.Application)
	model.Description = StringNullIfEmpty(webhookEndpoint.Description)
	enabledEvents, diags := types.ListValueFrom(ctx, types.StringType, webhookEndpoint.EnabledEvents)
	if diags.HasError() {
		respDiag.AddError(
			"Conversion Error",
			fmt.Sprintf("Error converting enabledEvents: %s", diags),
		)
		return
	}
	model.EnabledEvents = enabledEvents
	metadata, diags := types.MapValueFrom(ctx, types.StringType, webhookEndpoint.Metadata)
	if diags.HasError() {
		respDiag.AddError(
			"Conversion Error",
			fmt.Sprintf("Error converting metadata: %s", diags),
		)
		return
	}
	model.Metadata = MapValueNullIfEmpty(metadata, types.StringType)
	if webhookEndpoint.Status == "disabled" {
		model.Disabled = types.BoolValue(true)
	} else {
		model.Disabled = types.BoolValue(false)
	}
	model.URL = types.StringValue(webhookEndpoint.URL)
}

func (r *WebhookEndpointResource) buildCreateParams(plan WebhookEndpointResourceModel) *stripe.WebhookEndpointParams {
	params := &stripe.WebhookEndpointParams{}
	if !plan.APIVersion.IsNull() {
		params.APIVersion = plan.APIVersion.ValueStringPointer()
	}
	if !plan.Description.IsNull() {
		params.Description = plan.Description.ValueStringPointer()
	}
	if !plan.EnabledEvents.IsNull() {
		params.EnabledEvents = convertListToStringPtrs(plan.EnabledEvents)
	}
	if !plan.Metadata.IsNull() {
		for k, v := range plan.Metadata.Elements() {
			if str, ok := v.(types.String); ok {
				params.AddMetadata(k, str.ValueString())
			}
		}
	}
	if !plan.URL.IsNull() {
		params.URL = plan.URL.ValueStringPointer()
	}
	return params
}

func (r *WebhookEndpointResource) buildUpdateParams(state, plan WebhookEndpointResourceModel) *stripe.WebhookEndpointParams {
	params := &stripe.WebhookEndpointParams{}
	if !plan.Description.Equal(state.Description) {
		if plan.Description.IsNull() {
			params.Description = stripe.String("")
		} else {
			params.Description = plan.Description.ValueStringPointer()
		}
	}
	if !plan.Disabled.Equal(state.Disabled) {
		params.Disabled = plan.Disabled.ValueBoolPointer()
	}
	if !plan.EnabledEvents.Equal(state.EnabledEvents) {
		params.EnabledEvents = convertListToStringPtrs(plan.EnabledEvents)
	}
	if !plan.Metadata.Equal(state.Metadata) {
		planMetadata := plan.Metadata.Elements()
		stateMetadata := state.Metadata.Elements()
		for k, v := range planMetadata {
			if str, ok := v.(types.String); ok {
				params.AddMetadata(k, str.ValueString())
			}
		}
		for k := range stateMetadata {
			if _, exists := planMetadata[k]; !exists {
				params.AddMetadata(k, "")
			}
		}
	}
	if !plan.URL.Equal(state.URL) {
		params.URL = plan.URL.ValueStringPointer()
	}
	return params
}
