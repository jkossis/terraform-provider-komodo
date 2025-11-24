// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-komodo/internal/client"
)

// Ensure KomodoProvider satisfies various provider interfaces.
var _ provider.Provider = &KomodoProvider{}
var _ provider.ProviderWithFunctions = &KomodoProvider{}
var _ provider.ProviderWithEphemeralResources = &KomodoProvider{}

// KomodoProvider defines the provider implementation.
type KomodoProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// KomodoProviderModel describes the provider data model.
type KomodoProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (p *KomodoProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "komodo"
	resp.Version = p.version
}

func (p *KomodoProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for managing Komodo resources via the Komodo API.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The Komodo API endpoint URL. Can also be set via the KOMODO_ENDPOINT environment variable.",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The Komodo username. Can also be set via the KOMODO_USERNAME environment variable.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The Komodo password. Can also be set via the KOMODO_PASSWORD environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *KomodoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data KomodoProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check for environment variables if not set in config
	endpoint := data.Endpoint.ValueString()
	if endpoint == "" {
		endpoint = os.Getenv("KOMODO_ENDPOINT")
	}

	username := data.Username.ValueString()
	if username == "" {
		username = os.Getenv("KOMODO_USERNAME")
	}

	password := data.Password.ValueString()
	if password == "" {
		password = os.Getenv("KOMODO_PASSWORD")
	}

	// Validate required configuration
	if endpoint == "" {
		resp.Diagnostics.AddError(
			"Missing Komodo Endpoint",
			"The provider cannot create the Komodo API client as there is a missing or empty value for the Komodo endpoint. "+
				"Set the endpoint value in the configuration or use the KOMODO_ENDPOINT environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if username == "" {
		resp.Diagnostics.AddError(
			"Missing Komodo Username",
			"The provider cannot create the Komodo API client as there is a missing or empty value for the Komodo username. "+
				"Set the username value in the configuration or use the KOMODO_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddError(
			"Missing Komodo Password",
			"The provider cannot create the Komodo API client as there is a missing or empty value for the Komodo password. "+
				"Set the password value in the configuration or use the KOMODO_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create Komodo API client
	komodoClient := client.NewClient(endpoint, username, password)
	resp.DataSourceData = komodoClient
	resp.ResourceData = komodoClient
}

func (p *KomodoProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewApiKeyResource,
	}
}

func (p *KomodoProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *KomodoProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *KomodoProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &KomodoProvider{
			version: version,
		}
	}
}
