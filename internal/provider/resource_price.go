// Copyright (c) 2024 Zachary Koesters
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PriceResource{}
var _ resource.ResourceWithImportState = &PriceResource{}

func NewPriceResource() resource.Resource {
	return &PriceResource{}
}

// PriceResource defines the resource implementation.
type PriceResource struct {
	sc *client.API
}

// PriceResourceModel describes the resource data model.
type PriceResourceModel struct {
	Id                types.String  `tfsdk:"id"`
	Active            types.Bool    `tfsdk:"active"`
	BillingScheme     types.String  `tfsdk:"billing_scheme"`
	Currency          types.String  `tfsdk:"currency"`
	CurrencyOptions   types.Object  `tfsdk:"currency_options"`
	CustomUnitAmount  types.Object  `tfsdk:"custom_unit_amount"`
	LookupKey         types.String  `tfsdk:"lookup_key"`
	Metadata          types.Map     `tfsdk:"metadata"`
	Nickname          types.String  `tfsdk:"nickname"`
	Product           types.String  `tfsdk:"product"`
	Recurring         types.Object  `tfsdk:"recurring"`
	TaxBehavior       types.String  `tfsdk:"tax_behavior"`
	Tiers             types.List    `tfsdk:"tiers"`
	TiersMode         types.String  `tfsdk:"tiers_mode"`
	TransformQuantity types.Object  `tfsdk:"transform_quantity"`
	UnitAmount        types.Int64   `tfsdk:"unit_amount"`
	UnitAmountDecimal types.Float64 `tfsdk:"unit_amount_decimal"`
}

type PriceCustomUnitAmount struct {
	Maximum types.Int64 `tfsdk:"maximum"`
	Minimum types.Int64 `tfsdk:"minimum"`
	Preset  types.Int64 `tfsdk:"preset"`
}

type PriceCurrencyOptions struct {
	CustomUnitAmount  types.Object  `tfsdk:"custom_unit_amount"`
	TaxBehavior       types.String  `tfsdk:"tax_behavior"`
	Tiers             types.List    `tfsdk:"tiers"`
	UnitAmount        types.Int64   `tfsdk:"unit_amount"`
	UnitAmountDecimal types.Float64 `tfsdk:"unit_amount_decimal"`
	TopLevel          types.Bool    `tfsdk:"top_level"`
}

type PriceRecurring struct {
	Interval       types.String `tfsdk:"interval"`
	AggregateUsage types.String `tfsdk:"aggregate_usage"`
	IntervalCount  types.String `tfsdk:"interval_count"`
	Meter          types.String `tfsdk:"meter"`
	UsageType      types.String `tfsdk:"usage_type"`
}

type PriceTransformQuantity struct {
	DivideBy types.Int64  `tfsdk:"divide_by"`
	Round    types.String `tfsdk:"round"`
}

func (r *PriceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_price"
}

func (r *PriceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	customUnitAmountAttribute := schema.SingleNestedAttribute{
		MarkdownDescription: "When set, provides configuration for the amount to be adjusted by the customer during Checkout Sessions and Payment Links.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"maximum": schema.Int64Attribute{
				MarkdownDescription: "The maximum unit amount the customer can specify for this item.",
				Required:            true,
			},
			"minimum": schema.Int64Attribute{
				MarkdownDescription: "The minimum unit amount the customer can specify for this item. Must be at least the minimum charge amount.",
				Required:            true,
			},
			"preset": schema.Int64Attribute{
				MarkdownDescription: "The starting unit amount which can be updated by the customer.",
				Required:            true,
			},
		},
		Validators: []validator.Object{
			objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("unit_amount")),
			objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("unit_amount_decimal")),
		},
	}
	taxBehaviorAttribute := schema.StringAttribute{
		MarkdownDescription: "Specifies whether the price is considered inclusive of taxes or exclusive of taxes.",
		Computed:            true,
		Optional:            true,
		Default:             stringdefault.StaticString("unspecified"),
		Validators: []validator.String{
			stringvalidator.OneOf("exclusive", "inclusive", "unspecified"),
		},
	}
	tiersAttribute := schema.ListNestedAttribute{
		MarkdownDescription: "Each element represents a pricing tier. This parameter requires `billing_scheme` to be set to `tiered`. See also the documentation for `billing_scheme`.",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"flat_amount": schema.Int64Attribute{
					MarkdownDescription: "Price for the entire tier.",
					Optional:            true,
					Validators: []validator.Int64{
						int64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("flat_amount_decimal")),
					},
				},
				"flat_amount_decimal": schema.StringAttribute{
					MarkdownDescription: "Same as `flat_amount`, but contains a decimal value with at most 12 decimal places.",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("flat_amount")),
					},
				},
				"unit_amount": schema.Int64Attribute{
					MarkdownDescription: "Per unit price for units relevant to the tier.",
					Optional:            true,
					Validators: []validator.Int64{
						int64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("unit_amount_decimal")),
					},
				},
				"unit_amount_decimal": schema.StringAttribute{
					MarkdownDescription: "Same as `unit_amount`, but contains a decimal value with at most 12 decimal places.",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("unit_amount")),
					},
				},
				"up_to": schema.Int64Attribute{
					MarkdownDescription: "Up to and including to this quantity will be contained in the tier.",
					Required:            true,
					Validators: []validator.Int64{
						int64validator.AtLeast(0),
					},
				},
			},
		},
	}
	unitAmountAttribute := schema.Int64Attribute{
		MarkdownDescription: "The unit amount in cents to be charged, represented as a whole integer if possible. Only set if `billing_scheme=per_unit`.",
		Optional:            true,
		Validators: []validator.Int64{
			int64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("unit_amount_decimal")),
			int64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("custom_unit_amount")),
		},
	}
	unitAmountDecimalAttribute := schema.Float64Attribute{
		MarkdownDescription: "The unit amount in cents to be charged, represented as a decimal string with at most 12 decimal places. Only set if `billing_scheme=per_unit`.",
		Optional:            true,
		Validators: []validator.Float64{
			float64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("unit_amount")),
			float64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("custom_unit_amount")),
		},
	}
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A webhook endpoint resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the object",
				Computed:            true,
				Optional:            false,
				Required:            false,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether the price can be used for new purchases.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"billing_scheme": schema.StringAttribute{
				MarkdownDescription: "Describes how to compute the price per period. Either `per_unit` or `tiered`.",
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("per_unit"),
				Validators: []validator.String{
					stringvalidator.OneOf("per_unit", "tiered"),
				},
			},
			"currency": schema.StringAttribute{
				MarkdownDescription: "Three-letter ISO currency code, in lowercase. Must be a supported currency.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("currency_options")),
				},
			},
			"currency_options": schema.MapNestedAttribute{
				MarkdownDescription: "Prices defined in each available currency option.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"custom_unit_amount":  customUnitAmountAttribute,
						"tax_behavior":        taxBehaviorAttribute,
						"tiers":               tiersAttribute,
						"unit_amount":         unitAmountAttribute,
						"unit_amount_decimal": unitAmountDecimalAttribute,
						"top_level": schema.BoolAttribute{
							MarkdownDescription: "Whether the currency option is the top-level currency.",
							Computed:            true,
							Optional:            true,
							Default:             booldefault.StaticBool(false),
						},
					},
				},
				Optional: true,
				Validators: []validator.Map{
					mapvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("currency")),
					mapvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("unit_amount")),
					mapvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("unit_amount_decimal")),
				},
			},
			"lookup_key": schema.StringAttribute{
				MarkdownDescription: "A lookup key used to retrieve prices dynamically from a static string.",
				Optional:            true,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Set of key-value pairs that you can attach to an object. ",
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
			"nickname": schema.StringAttribute{
				MarkdownDescription: "A brief description of the price, hidden from customers.",
				Optional:            true,
			},
			"product": schema.StringAttribute{
				MarkdownDescription: "The ID of the product that this price will belong to.",
				Required:            true,
			},
			"recurring": schema.SingleNestedAttribute{
				MarkdownDescription: "The recurring components of a price such as `interval` and `usage_type`.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"interval": schema.StringAttribute{
						MarkdownDescription: "Specifies billing frequency. Either `day`, `week`, `month` or `year`.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("day", "week", "month", "year"),
						},
					},
					"aggregate_usage": schema.StringAttribute{
						MarkdownDescription: "Specifies a usage aggregation strategy for prices of `usage_type=metered`. Defaults to `sum`.",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("sum"),
						Validators: []validator.String{
							stringvalidator.OneOf("last_during_period", "last_ever", "max", "sum"),
						},
					},
					"interval_count": schema.StringAttribute{
						MarkdownDescription: "The number of intervals (specified in the `interval` attribute) between subscription billings.",
						Optional:            true,
					},
					"meter": schema.StringAttribute{
						MarkdownDescription: "The meter tracking the usage of a metered price.",
						Optional:            true,
					},
					"usage_type": schema.StringAttribute{
						MarkdownDescription: "Configures how the quantity per period should be determined.",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("licensed"),
						Validators: []validator.String{
							stringvalidator.OneOf("licensed", "metered"),
						},
					},
				},
			},
			"tiers_mode": schema.StringAttribute{
				MarkdownDescription: "Defines if the tiering price should be `graduated` or `volume` based. In `volume`-based tiering, the maximum quantity within a period determines the per unit price. In `graduated` tiering, pricing can change as the quantity grows.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("graduated", "volume"),
				},
			},
			"transform_quantity": schema.SingleNestedAttribute{
				MarkdownDescription: "Apply a transformation to the reported usage or set quantity before computing the amount billed. Cannot be combined with `tiers`.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"divide_by": schema.Int64Attribute{
						MarkdownDescription: "Divide usage by this number.",
						Required:            true,
					},
					"round": schema.StringAttribute{
						MarkdownDescription: "After division, either round the result `up` or `down`.",
						Required:            true,
					},
				},
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("tiers")),
				},
			},
		},
	}
}

func (r *PriceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PriceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PriceResourceModel
	var price *stripe.Price
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

	price, err = r.sc.Prices.New(params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create price, got error: %s", err))
		return
	}

	plan.Id = types.StringValue(price.ID)
	r.populateModel(&plan, price)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PriceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PriceResourceModel
	var price *stripe.Price
	var err error

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	price, err = r.sc.Prices.Get(state.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read price, got error: %s", err))
		return
	}

	r.populateModel(&state, price)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PriceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan PriceResourceModel
	var price *stripe.Price
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

	price, err = r.sc.Prices.Update(plan.Id.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create price, got error: %s", err))
		return
	}
	r.populateModel(&plan, price)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PriceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PriceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddError("Client Error", "Stripe API does not support deleting prices. Please archive the price instead.")

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *PriceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var state PriceResourceModel
	var price *stripe.Price
	var err error

	price, err = r.sc.Prices.Get(req.ID, nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import price, got error: %s", err))
		return
	}

	state.Id = types.StringValue(req.ID)
	r.populateModel(&state, price)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PriceResource) populateModel(model *PriceResourceModel, price *stripe.Price) {
	model.Active = types.BoolValue(price.Active)
	model.BillingScheme = types.StringValue(string(price.BillingScheme))
	model.Currency = types.StringValue(string(price.Currency))
	model.LookupKey = types.StringValue(price.LookupKey)
	model.Nickname = types.StringValue(price.Nickname)
	model.Product = types.StringValue(price.Product.ID)
	model.TaxBehavior = types.StringValue(string(price.TaxBehavior))
	model.Tiers = types.List{}
	model.TiersMode = types.StringValue(string(price.TiersMode))
	model.UnitAmount = types.Int64Value(price.UnitAmount)
	model.UnitAmountDecimal = types.Float64Value(price.UnitAmountDecimal)
}

func (r *PriceResource) buildCreateParams(plan PriceResourceModel) *stripe.PriceParams {
	params := &stripe.PriceParams{}
	return params
}

func (r *PriceResource) buildUpdateParams(state, plan PriceResourceModel) *stripe.PriceParams {
	params := &stripe.PriceParams{}
	return params
}
