package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
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
var _ resource.Resource = &CouponResource{}
var _ resource.ResourceWithImportState = &CouponResource{}

func NewCouponResource() resource.Resource {
	return &CouponResource{}
}

// CouponResource defines the resource implementation.
type CouponResource struct {
	sc *client.API
}

// CouponResourceModel describes the resource data model.
type CouponResourceModel struct {
	Id               types.String  `tfsdk:"id"`
	AppliesTo        types.List    `tfsdk:"applies_to"`
	CurrencyOptions  types.Map     `tfsdk:"currency_options"`
	Duration         types.String  `tfsdk:"duration"`
	DurationInMonths types.Int64   `tfsdk:"duration_in_months"`
	MaxRedemptions   types.Int64   `tfsdk:"max_redemptions"`
	Metadata         types.Map     `tfsdk:"metadata"`
	Name             types.String  `tfsdk:"name"`
	PercentOff       types.Float64 `tfsdk:"percent_off"`
	RedeemBy         types.Int64   `tfsdk:"redeem_by"`
}

type CouponCurrencyOptionsModel struct {
	AmountOff types.Int64 `tfsdk:"amount_off"`
	TopLevel  types.Bool  `tfsdk:"top_level"`
}

func (m CouponCurrencyOptionsModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"amount_off": types.Int64Type,
		"top_level":  types.BoolType,
	}
}

func (r *CouponResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_coupon"
}

func (r *CouponResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A webhook endpoint resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the object.",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"applies_to": schema.ListAttribute{
				MarkdownDescription: "An array of Product IDs that this Coupon will apply to.",
				ElementType:         types.StringType,
				Optional:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				Validators: []validator.List{
					listvalidator.UniqueValues(),
				},
			},
			"currency_options": schema.MapNestedAttribute{
				MarkdownDescription: "Coupons defined in each available currency option. Each key must be a three-letter ISO currency code and a supported currency.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"amount_off": schema.Int64Attribute{
							MarkdownDescription: "Amount (in the `currency` specified) that will be taken off the subtotal of any invoices for this customer.",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
						},
						"top_level": schema.BoolAttribute{
							MarkdownDescription: "Whether the currency option is the top-level currency.",
							Computed:            true,
							Optional:            true,
							Default:             booldefault.StaticBool(false),
						},
					},
				},
				Optional: true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplaceIf(
						func(ctx context.Context, request planmodifier.MapRequest, response *mapplanmodifier.RequiresReplaceIfFuncResponse) {
							if request.PlanValue.Equal(request.StateValue) {
								return
							}
							planCurrencyOptions := map[string]CouponCurrencyOptionsModel{}
							stateCurrencyOptions := map[string]CouponCurrencyOptionsModel{}
							request.PlanValue.ElementsAs(ctx, &planCurrencyOptions, false)
							request.StateValue.ElementsAs(ctx, &stateCurrencyOptions, false)
							for k, v := range planCurrencyOptions {
								if _, exists := stateCurrencyOptions[k]; exists {
									if stateCurrencyOptions[k].AmountOff != v.AmountOff {
										response.RequiresReplace = true
									}
									if stateCurrencyOptions[k].TopLevel != v.TopLevel {
										response.RequiresReplace = true
									}
								}
							}
							for k := range stateCurrencyOptions {
								if _, exists := planCurrencyOptions[k]; !exists {
									response.RequiresReplace = true
								}
							}
						},
						"If values of elements are change or elements are removed, Terraform will destroy and recreate the resource.",
						"If values of elements are change or elements are removed, Terraform will destroy and recreate the resource.",
					),
				},
				Validators: []validator.Map{
					mapvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("percent_off")),
				},
			},
			"duration": schema.StringAttribute{
				MarkdownDescription: "One of `forever`, `once`, and `repeating`. Describes how long a customer who applies this coupon will get the discount.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("once"),
				Validators: []validator.String{
					stringvalidator.OneOf("forever", "once", "repeating"),
				},
			},
			"duration_in_months": schema.Int64Attribute{
				MarkdownDescription: "If duration is `repeating`, the number of months the coupon applies. Null if coupon duration is forever or once.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AlsoRequires(path.MatchRelative().AtParent().AtName("duration")),
				},
			},
			"max_redemptions": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of times this coupon can be redeemed, in total, across all customers, before it is no longer valid.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
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
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the coupon displayed to customers on for instance invoices or receipts.",
				Optional:            true,
			},
			"percent_off": schema.Float64Attribute{
				MarkdownDescription: "Percent that will be taken off the subtotal of any invoices for this customer for the duration of the coupon.",
				Optional:            true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Float64{
					float64validator.Between(1, 100),
					float64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("currency_options")),
				},
			},
			"redeem_by": schema.Int64Attribute{
				MarkdownDescription: "Date after which the coupon can no longer be redeemed.",
				Optional:            true,
			},
		},
	}
}

func (r *CouponResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CouponResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CouponResourceModel
	var coupon *stripe.Coupon
	var err error

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := r.buildCreateParams(ctx, plan, resp.Diagnostics)
	params.AddExpand("currency_options")
	coupon, err = r.sc.Coupons.New(params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create webhook endpoint, got error: %s", err))
		return
	}

	plan.Id = types.StringValue(coupon.ID)
	r.populateModel(ctx, &plan, coupon, resp.Diagnostics)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CouponResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CouponResourceModel
	var coupon *stripe.Coupon
	var err error

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := &stripe.CouponParams{}
	params.AddExpand("currency_options")
	coupon, err = r.sc.Coupons.Get(state.Id.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read webhook endpoint, got error: %s", err))
		return
	}

	r.populateModel(ctx, &state, coupon, resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CouponResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan CouponResourceModel
	var coupon *stripe.Coupon
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

	params := r.buildUpdateParams(ctx, state, plan, resp.Diagnostics)
	params.AddExpand("currency_options")
	coupon, err = r.sc.Coupons.Update(plan.Id.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create webhook endpoint, got error: %s", err))
		return
	}
	r.populateModel(ctx, &plan, coupon, resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CouponResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CouponResourceModel
	var err error

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err = r.sc.Coupons.Del(state.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete webhook endpoint, got error: %s", err))
		return
	}
}

func (r *CouponResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var state CouponResourceModel
	var coupon *stripe.Coupon
	var err error

	params := &stripe.CouponParams{}
	params.AddExpand("currency_options")
	coupon, err = r.sc.Coupons.Get(req.ID, params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import webhook endpoint, got error: %s", err))
		return
	}

	state.Id = types.StringValue(req.ID)
	r.populateModel(ctx, &state, coupon, resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CouponResource) populateModel(ctx context.Context, model *CouponResourceModel, coupon *stripe.Coupon, respDiag diag.Diagnostics) {
	if coupon.AppliesTo != nil && coupon.AppliesTo.Products != nil {
		appliesTo, diags := types.ListValueFrom(ctx, types.StringType, coupon.AppliesTo.Products)
		if diags.HasError() {
			respDiag.Append(diags...)
		}
		model.AppliesTo = ListValueNullIfEmpty(appliesTo, types.StringType)
	} else {
		model.AppliesTo = types.ListNull(types.StringType)
	}

	currencyOptions := map[string]CouponCurrencyOptionsModel{}
	for currency, cco := range coupon.CurrencyOptions {
		ccom := CouponCurrencyOptionsModel{}
		ccom.AmountOff = Int64NullIfEmpty(cco.AmountOff)
		if string(coupon.Currency) == currency {
			ccom.TopLevel = types.BoolValue(true)
		} else {
			ccom.TopLevel = types.BoolValue(false)
		}
		currencyOptions[currency] = ccom
	}
	t, diags := types.MapValueFrom(
		ctx,
		types.ObjectType{
			AttrTypes: CouponCurrencyOptionsModel{}.Types(),
		},
		currencyOptions,
	)
	if diags.HasError() {
		respDiag.Append(diags...)
	}
	model.CurrencyOptions = MapValueNullIfEmpty(t, types.ObjectType{
		AttrTypes: CouponCurrencyOptionsModel{}.Types(),
	})
	model.Duration = StringNullIfEmpty(string(coupon.Duration))
	model.DurationInMonths = Int64NullIfEmpty(coupon.DurationInMonths)
	model.MaxRedemptions = Int64NullIfEmpty(coupon.MaxRedemptions)
	metadata, diags := types.MapValueFrom(ctx, types.StringType, coupon.Metadata)
	if diags.HasError() {
		respDiag.Append(diags...)
	}
	model.Metadata = MapValueNullIfEmpty(metadata, types.StringType)
	model.Name = StringNullIfEmpty(coupon.Name)
	model.PercentOff = Float64NullIfEmpty(coupon.PercentOff)
	model.RedeemBy = Int64NullIfEmpty(coupon.RedeemBy)
}

func (r *CouponResource) buildCreateParams(ctx context.Context, data CouponResourceModel, respDiag diag.Diagnostics) *stripe.CouponParams {
	params := &stripe.CouponParams{}
	if !data.Id.IsNull() && !data.Id.IsUnknown() {
		params.ID = data.Id.ValueStringPointer()
	}
	if !data.AppliesTo.IsUnknown() && !data.AppliesTo.IsNull() {
		cat := &stripe.CouponAppliesToParams{}
		for _, v := range data.AppliesTo.Elements() {
			if str, ok := v.(types.String); ok {
				cat.Products = append(cat.Products, str.ValueStringPointer())
			}
		}
		params.AppliesTo = cat
	}
	if !data.CurrencyOptions.IsUnknown() && !data.CurrencyOptions.IsNull() {
		currencyOptions := map[string]CouponCurrencyOptionsModel{}
		params.CurrencyOptions = map[string]*stripe.CouponCurrencyOptionsParams{}
		diags := data.CurrencyOptions.ElementsAs(ctx, &currencyOptions, false)
		if diags.HasError() {
			respDiag.Append(diags...)
		}
		for key, element := range currencyOptions {
			if element.TopLevel.ValueBool() {
				params.AmountOff = element.AmountOff.ValueInt64Pointer()
				params.Currency = stripe.String(key)
			} else {
				cco := &stripe.CouponCurrencyOptionsParams{
					AmountOff: element.AmountOff.ValueInt64Pointer(),
				}
				params.CurrencyOptions[key] = cco
			}
		}
	}
	if !data.Duration.IsUnknown() {
		params.Duration = data.Duration.ValueStringPointer()
	}
	if !data.DurationInMonths.IsUnknown() {
		params.DurationInMonths = data.DurationInMonths.ValueInt64Pointer()
	}
	if !data.Metadata.IsUnknown() {
		for k, v := range data.Metadata.Elements() {
			if str, ok := v.(types.String); ok {
				params.AddMetadata(k, str.ValueString())
			}
		}
	}
	if !data.MaxRedemptions.IsUnknown() {
		params.MaxRedemptions = data.MaxRedemptions.ValueInt64Pointer()
	}
	if !data.Name.IsUnknown() {
		params.Name = data.Name.ValueStringPointer()
	}
	if !data.PercentOff.IsUnknown() {
		params.PercentOff = data.PercentOff.ValueFloat64Pointer()
	}
	if !data.RedeemBy.IsUnknown() {
		params.RedeemBy = data.RedeemBy.ValueInt64Pointer()
	}
	return params
}

func (r *CouponResource) buildUpdateParams(ctx context.Context, state, plan CouponResourceModel, respDiag diag.Diagnostics) *stripe.CouponParams {
	params := &stripe.CouponParams{}

	if !plan.CurrencyOptions.Equal(state.CurrencyOptions) {
		params.CurrencyOptions = map[string]*stripe.CouponCurrencyOptionsParams{}
		stateCurrencyOptions := map[string]CouponCurrencyOptionsModel{}
		diags := state.CurrencyOptions.ElementsAs(ctx, &stateCurrencyOptions, false)
		if diags.HasError() {
			respDiag.Append(diags...)
		}
		planCurrencyOptions := map[string]CouponCurrencyOptionsModel{}
		diags = plan.CurrencyOptions.ElementsAs(ctx, &planCurrencyOptions, false)
		if diags.HasError() {
			respDiag.Append(diags...)
		}
		for k, v := range planCurrencyOptions {
			if _, exists := stateCurrencyOptions[k]; !exists {
				params.CurrencyOptions[k] = &stripe.CouponCurrencyOptionsParams{
					AmountOff: v.AmountOff.ValueInt64Pointer(),
				}
			}
		}
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

	if !plan.Name.Equal(state.Name) {
		params.Name = EmptyStringIfNull(plan.Name)
	}

	return params
}
