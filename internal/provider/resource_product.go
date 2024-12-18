package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProductResource{}
var _ resource.ResourceWithImportState = &ProductResource{}

func NewProductResource() resource.Resource {
	return &ProductResource{}
}

// ProductResource defines the resource implementation.
type ProductResource struct {
	sc *client.API
}

// ProductResourceModel describes the resource data model.
type ProductResourceModel struct {
	Id                  types.String `tfsdk:"id"`
	Active              types.Bool   `tfsdk:"active"`
	DefaultPrice        types.String `tfsdk:"default_price"`
	Description         types.String `tfsdk:"description"`
	Images              types.List   `tfsdk:"images"`
	MarketingFeatures   types.List   `tfsdk:"marketing_features"`
	Metadata            types.Map    `tfsdk:"metadata"`
	Name                types.String `tfsdk:"name"`
	PackageDimensions   types.Object `tfsdk:"package_dimensions"`
	Shippable           types.Bool   `tfsdk:"shippable"`
	StatementDescriptor types.String `tfsdk:"statement_descriptor"`
	TaxCode             types.String `tfsdk:"tax_code"`
	UnitLabel           types.String `tfsdk:"unit_label"`
	URL                 types.String `tfsdk:"url"`
}

// ProductPackageDimensionsResourceModel represents the dimensions of a product package including height, length, weight, and width.
type ProductPackageDimensionsResourceModel struct {
	Height types.Float64 `tfsdk:"height"`
	Length types.Float64 `tfsdk:"length"`
	Weight types.Float64 `tfsdk:"weight"`
	Width  types.Float64 `tfsdk:"width"`
}

func (m ProductPackageDimensionsResourceModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"height": types.Float64Type,
		"length": types.Float64Type,
		"weight": types.Float64Type,
		"width":  types.Float64Type,
	}
}

func (r *ProductResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product"
}

func (r *ProductResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Products describe the specific goods or services you offer to your customers.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the object",
				Computed:            true,
				Required:            false,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether the product is currently available for purchase.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"default_price": schema.StringAttribute{
				MarkdownDescription: "The ID of the Price object that is the default price for this product.",
				Required:            false,
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The product’s description, meant to be displayable to the customer.",
				Required:            false,
				Optional:            true,
			},
			"images": schema.ListAttribute{
				MarkdownDescription: "A list of up to 8 URLs of images for this product, meant to be displayable to the customer.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators: []validator.List{
					listvalidator.UniqueValues(),
					listvalidator.SizeAtMost(8),
				},
			},
			"marketing_features": schema.ListAttribute{
				MarkdownDescription: "A list of up to 15 marketing features for this product. These are displayed in pricing tables.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators: []validator.List{
					listvalidator.UniqueValues(),
					listvalidator.SizeAtMost(15),
					listvalidator.ValueStringsAre(stringvalidator.LengthAtMost(80)),
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
				MarkdownDescription: "The product’s name, meant to be displayable to the customer.",
				Required:            true,
			},
			"package_dimensions": schema.SingleNestedAttribute{
				MarkdownDescription: "The dimensions of this product for shipping purposes.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"height": schema.Float64Attribute{
						MarkdownDescription: "Height, in inches.",
						Required:            true,
					},
					"length": schema.Float64Attribute{
						MarkdownDescription: "Length, in inches.",
						Required:            true,
					},
					"weight": schema.Float64Attribute{
						MarkdownDescription: "Weight, in ounces.",
						Required:            true,
					},
					"width": schema.Float64Attribute{
						MarkdownDescription: "Width, in inches.",
						Required:            true,
					},
				},
			},
			"shippable": schema.BoolAttribute{
				MarkdownDescription: "Whether this product is shipped (i.e., physical goods).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"statement_descriptor": schema.StringAttribute{
				MarkdownDescription: "Extra information about a product which will appear on your customer’s credit card statement.",
				Optional:            true,
			},
			"tax_code": schema.StringAttribute{
				MarkdownDescription: "A tax code ID.",
				Optional:            true,
			},
			"unit_label": schema.StringAttribute{
				MarkdownDescription: "A label that represents units of this product. When set, this will be included in customers’ receipts, invoices, Checkout, and the customer portal.",
				Optional:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "A URL of a publicly-accessible webpage for this product.",
				Optional:            true,
			},
		},
	}
}

func (r *ProductResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProductResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProductResourceModel
	var product *stripe.Product
	var err error

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := r.buildCreateParams(ctx, plan, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	product, err = r.sc.Products.New(params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create webhook endpoint, got error: %s", err))
		return
	}

	plan.Id = types.StringValue(product.ID)
	r.populateModel(ctx, &plan, product, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProductResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProductResourceModel
	var product *stripe.Product
	var err error

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	product, err = r.sc.Products.Get(state.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read webhook endpoint, got error: %s", err))
		return
	}

	r.populateModel(ctx, &state, product, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProductResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan ProductResourceModel
	var product *stripe.Product
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
	if resp.Diagnostics.HasError() {
		return
	}

	product, err = r.sc.Products.Update(plan.Id.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create webhook endpoint, got error: %s", err))
		return
	}

	r.populateModel(ctx, &plan, product, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProductResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProductResourceModel
	var err error

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err = r.sc.Products.Del(state.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete webhook endpoint, got error: %s", err))
		return
	}
}

func (r *ProductResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var state ProductResourceModel
	var product *stripe.Product
	var err error

	product, err = r.sc.Products.Get(req.ID, nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import webhook endpoint, got error: %s", err))
		return
	}

	state.Id = types.StringValue(req.ID)
	r.populateModel(ctx, &state, product, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProductResource) populateModel(ctx context.Context, model *ProductResourceModel, product *stripe.Product, respDiag diag.Diagnostics) {
	model.Active = types.BoolValue(product.Active)
	if product.DefaultPrice != nil {
		model.DefaultPrice = types.StringValue(product.DefaultPrice.ID)

	}
	model.Description = StringNullIfEmpty(product.Description)
	images, diags := types.ListValueFrom(ctx, types.StringType, product.Images)
	if diags.HasError() {
		respDiag.Append(diags...)
	}
	model.Images = ListValueNullIfEmpty(images, types.StringType)
	if product.MarketingFeatures != nil {
		var marketingFeatures []string
		for _, v := range product.MarketingFeatures {
			marketingFeatures = append(marketingFeatures, v.Name)
		}
		m, diags := types.ListValueFrom(ctx, types.StringType, marketingFeatures)
		if diags.HasError() {
			respDiag.Append(diags...)
		}
		model.MarketingFeatures = ListValueNullIfEmpty(m, types.StringType)
	}
	metadata, diags := types.MapValueFrom(ctx, types.StringType, product.Metadata)
	if diags.HasError() {
		respDiag.Append(diags...)
	}
	model.Metadata = MapValueNullIfEmpty(metadata, types.StringType)
	model.Name = types.StringValue(product.Name)
	if product.PackageDimensions != nil && product.PackageDimensions.Height != 0 && product.PackageDimensions.Length != 0 && product.PackageDimensions.Weight != 0 && product.PackageDimensions.Width != 0 {
		p, diags := types.ObjectValueFrom(
			ctx,
			ProductPackageDimensionsResourceModel{}.Types(),
			&ProductPackageDimensionsResourceModel{
				Height: types.Float64Value(product.PackageDimensions.Height),
				Length: types.Float64Value(product.PackageDimensions.Length),
				Weight: types.Float64Value(product.PackageDimensions.Weight),
				Width:  types.Float64Value(product.PackageDimensions.Width),
			},
		)
		if diags.HasError() {
			respDiag.Append(diags...)
		}
		model.PackageDimensions = p
	} else {
		model.PackageDimensions = types.ObjectNull(ProductPackageDimensionsResourceModel{}.Types())
	}
	model.Shippable = types.BoolValue(product.Shippable)
	model.StatementDescriptor = StringNullIfEmpty(product.StatementDescriptor)
	if product.TaxCode != nil {
		model.TaxCode = types.StringValue(product.TaxCode.ID)

	}
	model.UnitLabel = StringNullIfEmpty(product.UnitLabel)
	model.URL = StringNullIfEmpty(product.URL)
}

func (r *ProductResource) buildCreateParams(ctx context.Context, plan ProductResourceModel, respDiag diag.Diagnostics) *stripe.ProductParams {
	params := &stripe.ProductParams{}
	if !plan.Id.IsUnknown() {
		params.ID = plan.Id.ValueStringPointer()
	}
	if !plan.Active.IsUnknown() {
		params.Active = plan.Active.ValueBoolPointer()
	}
	if !plan.DefaultPrice.IsUnknown() {
		params.DefaultPrice = plan.DefaultPrice.ValueStringPointer()
	}
	if !plan.Description.IsUnknown() {
		params.Description = plan.Description.ValueStringPointer()
	}
	if !plan.Images.IsUnknown() {
		params.Images = convertListToStringPtrs(plan.Images)
	}
	if !plan.MarketingFeatures.IsUnknown() && !plan.MarketingFeatures.IsNull() {
		params.MarketingFeatures = []*stripe.ProductMarketingFeatureParams{}
		for _, v := range plan.MarketingFeatures.Elements() {
			if str, ok := v.(types.String); ok {
				pmf := &stripe.ProductMarketingFeatureParams{
					Name: str.ValueStringPointer(),
				}
				params.MarketingFeatures = append(params.MarketingFeatures, pmf)
			}
		}
	}
	if !plan.Metadata.IsNull() {
		for k, v := range plan.Metadata.Elements() {
			if str, ok := v.(types.String); ok {
				params.AddMetadata(k, str.ValueString())
			}
		}
	}
	if !plan.Name.IsUnknown() {
		params.Name = plan.Name.ValueStringPointer()
	}
	if !plan.PackageDimensions.IsUnknown() && !plan.PackageDimensions.IsNull() {
		packageDimensions := ProductPackageDimensionsResourceModel{}
		diags := plan.PackageDimensions.As(ctx, &packageDimensions, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    false,
			UnhandledUnknownAsEmpty: false,
		})
		if diags.HasError() {
			respDiag.Append(diags...)
		}
		params.PackageDimensions = &stripe.ProductPackageDimensionsParams{
			Height: packageDimensions.Height.ValueFloat64Pointer(),
			Length: packageDimensions.Length.ValueFloat64Pointer(),
			Weight: packageDimensions.Weight.ValueFloat64Pointer(),
			Width:  packageDimensions.Width.ValueFloat64Pointer(),
		}
	}
	if !plan.Shippable.IsUnknown() {
		params.Shippable = plan.Shippable.ValueBoolPointer()
	}
	if !plan.StatementDescriptor.IsUnknown() {
		params.StatementDescriptor = plan.StatementDescriptor.ValueStringPointer()
	}
	if !plan.TaxCode.IsUnknown() {
		params.TaxCode = plan.TaxCode.ValueStringPointer()
	}
	if !plan.UnitLabel.IsUnknown() {
		params.UnitLabel = plan.UnitLabel.ValueStringPointer()
	}
	if !plan.URL.IsUnknown() {
		params.URL = plan.URL.ValueStringPointer()
	}
	return params
}

func (r *ProductResource) buildUpdateParams(ctx context.Context, state, plan ProductResourceModel, respDiag diag.Diagnostics) *stripe.ProductParams {
	params := &stripe.ProductParams{}
	if !plan.Active.Equal(state.Active) {
		params.Active = plan.Active.ValueBoolPointer()
	}
	if !plan.DefaultPrice.Equal(state.DefaultPrice) {
		params.DefaultPrice = EmptyStringIfNull(plan.DefaultPrice)
	}
	if !plan.Description.Equal(state.Description) {
		params.Description = EmptyStringIfNull(plan.Description)
	}
	if !plan.Images.Equal(state.Images) {
		params.Images = convertListToStringPtrs(plan.Images)
	}
	if !plan.MarketingFeatures.Equal(state.MarketingFeatures) {
		params.MarketingFeatures = []*stripe.ProductMarketingFeatureParams{}
		for _, v := range plan.MarketingFeatures.Elements() {
			if str, ok := v.(types.String); ok {
				pmf := &stripe.ProductMarketingFeatureParams{
					Name: str.ValueStringPointer(),
				}
				params.MarketingFeatures = append(params.MarketingFeatures, pmf)
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
		params.Name = plan.Name.ValueStringPointer()
	}
	if !plan.PackageDimensions.Equal(state.PackageDimensions) {
		if plan.PackageDimensions.IsNull() {
			params.PackageDimensions = &stripe.ProductPackageDimensionsParams{
				Height: stripe.Float64(0),
				Length: stripe.Float64(0),
				Weight: stripe.Float64(0),
				Width:  stripe.Float64(0),
			}
		} else {
			packageDimensions := ProductPackageDimensionsResourceModel{}
			diags := plan.PackageDimensions.As(ctx, &packageDimensions, basetypes.ObjectAsOptions{
				UnhandledNullAsEmpty:    false,
				UnhandledUnknownAsEmpty: false,
			})
			if diags.HasError() {
				respDiag.Append(diags...)
			}
			params.PackageDimensions = &stripe.ProductPackageDimensionsParams{
				Height: packageDimensions.Height.ValueFloat64Pointer(),
				Length: packageDimensions.Length.ValueFloat64Pointer(),
				Weight: packageDimensions.Weight.ValueFloat64Pointer(),
				Width:  packageDimensions.Width.ValueFloat64Pointer(),
			}
		}
	}
	if !plan.Shippable.Equal(state.Shippable) {
		params.Shippable = plan.Shippable.ValueBoolPointer()
	}
	if !plan.StatementDescriptor.Equal(state.StatementDescriptor) {
		params.StatementDescriptor = EmptyStringIfNull(plan.StatementDescriptor)
	}
	if !plan.TaxCode.Equal(state.TaxCode) {
		params.TaxCode = EmptyStringIfNull(plan.TaxCode)
	}
	if !plan.UnitLabel.Equal(state.UnitLabel) {
		params.UnitLabel = EmptyStringIfNull(plan.UnitLabel)
	}
	if !plan.URL.Equal(state.URL) {
		params.URL = EmptyStringIfNull(plan.URL)
	}
	return params
}
