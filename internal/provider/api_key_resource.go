// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jkossis/terraform-provider-komodo/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &APIKeyResource{}
var _ resource.ResourceWithImportState = &APIKeyResource{}

func NewAPIKeyResource() resource.Resource {
	return &APIKeyResource{}
}

// APIKeyResource defines the resource implementation.
type APIKeyResource struct {
	client *client.Client
}

// APIKeyResourceModel describes the resource data model.
type APIKeyResourceModel struct {
	Key       types.String `tfsdk:"key"`
	Secret    types.String `tfsdk:"secret"`
	Name      types.String `tfsdk:"name"`
	UserID    types.String `tfsdk:"user_id"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	Expires   types.Int64  `tfsdk:"expires"`
}

func (r *APIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = typeNamePrefix + "_api_key"
}

func (r *APIKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Komodo API key.",

		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The API key identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The API key secret (only available on creation).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A human-friendly name for the API key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					nonEmptyStringValidator{description: "API key name must not be empty"},
				},
			},
			"user_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the user who owns this key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp in milliseconds since epoch.",
			},
			"expires": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "Expiration timestamp in milliseconds since epoch. Use 0 for no expiration.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64AtLeastValidator{min: 0, description: "API key expiration must be greater than or equal to 0"},
				},
			},
		},
	}
}

func (r *APIKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data APIKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating API key", map[string]interface{}{
		"name":    data.Name.ValueString(),
		"expires": data.Expires.ValueInt64(),
	})

	createReq := client.CreateAPIKeyRequest{
		Name:    data.Name.ValueString(),
		Expires: data.Expires.ValueInt64(),
	}

	key, err := r.client.CreateAPIKey(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create API key, got error: %s", err))
		return
	}

	data.Key = types.StringValue(key.Key)
	data.Secret = types.StringValue(key.Secret)
	data.UserID = types.StringValue(key.UserID)
	data.Name = types.StringValue(key.Name)
	data.CreatedAt = types.Int64Value(key.CreatedAt)
	data.Expires = types.Int64Value(key.Expires)

	tflog.Trace(ctx, "Created API key resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data APIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	keyID := data.Key.ValueString()
	key, err := r.client.GetAPIKey(ctx, keyID)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read API key, got error: %s", err))
		return
	}

	if key == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with key information
	data.Key = types.StringValue(key.Key)
	data.UserID = types.StringValue(key.UserID)
	data.Name = types.StringValue(key.Name)
	data.CreatedAt = types.Int64Value(key.CreatedAt)
	data.Expires = types.Int64Value(key.Expires)
	// Secret is not returned by GetAPIKey (only on creation), so retain the existing value.

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data APIKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configurable attributes require replacement, so Update has no API operation.
	tflog.Trace(ctx, "API key resource update is a no-op")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data APIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting API key", map[string]interface{}{
		"key": data.Key.ValueString(),
	})

	err := r.client.DeleteAPIKey(ctx, client.DeleteAPIKeyRequest{
		Key: data.Key.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete API key, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "Deleted API key resource")
}

func (r *APIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("key"), req, resp)
}
