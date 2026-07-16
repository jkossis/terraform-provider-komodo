// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jkossis/terraform-provider-komodo/internal/client"
)

const (
	providerTypeName  = "komodo"
	typeNamePrefix    = providerTypeName
	komodoEndpointEnv = "KOMODO_ENDPOINT"
	komodoUsernameEnv = "KOMODO_USERNAME"
	komodoPasswordEnv = "KOMODO_PASSWORD"
)

// Ensure KomodoProvider satisfies various provider interfaces.
var _ provider.Provider = &KomodoProvider{}

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

type providerConfig struct {
	endpoint string
	username string
	password string
}

type providerStringConfigSource struct {
	value       types.String
	attribute   string
	envVar      string
	displayName string
}

func (p *KomodoProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = providerTypeName
	resp.Version = p.version
}

func (p *KomodoProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for managing Komodo resources via the Komodo API.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The Komodo API endpoint URL. Can also be set via the " + komodoEndpointEnv + " environment variable.",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The Komodo username. Can also be set via the " + komodoUsernameEnv + " environment variable.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The Komodo password. Can also be set via the " + komodoPasswordEnv + " environment variable.",
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

	config, diags := providerConfigFrom(data, os.Getenv)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	komodoClient := client.NewClient(config.endpoint, config.username, config.password)
	resp.DataSourceData = komodoClient
	resp.ResourceData = komodoClient
}

func providerConfigFrom(data KomodoProviderModel, lookupEnv func(string) string) (providerConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	config := providerConfig{
		endpoint: stringConfigValue(providerStringConfigSource{
			value:       data.Endpoint,
			attribute:   "endpoint",
			envVar:      komodoEndpointEnv,
			displayName: "Komodo Endpoint",
		}, lookupEnv, &diags),
		username: stringConfigValue(providerStringConfigSource{
			value:       data.Username,
			attribute:   "username",
			envVar:      komodoUsernameEnv,
			displayName: "Komodo Username",
		}, lookupEnv, &diags),
		password: stringConfigValue(providerStringConfigSource{
			value:       data.Password,
			attribute:   "password",
			envVar:      komodoPasswordEnv,
			displayName: "Komodo Password",
		}, lookupEnv, &diags),
	}

	return config, diags
}

func stringConfigValue(source providerStringConfigSource, lookupEnv func(string) string, diags *diag.Diagnostics) string {
	if source.value.IsUnknown() {
		addUnknownProviderConfigDiagnostic(source, diags)
		return ""
	}

	if !source.value.IsNull() {
		value := source.value.ValueString()
		if value == "" {
			addMissingProviderConfigDiagnostic(source, diags)
		}
		return value
	}

	value := lookupEnv(source.envVar)
	if value == "" {
		addMissingProviderConfigDiagnostic(source, diags)
	}
	return value
}

func addUnknownProviderConfigDiagnostic(source providerStringConfigSource, diags *diag.Diagnostics) {
	diags.AddAttributeError(
		path.Root(source.attribute),
		"Unknown "+source.displayName,
		"The "+source.attribute+" provider attribute must be known during provider configuration.",
	)
}

func addMissingProviderConfigDiagnostic(source providerStringConfigSource, diags *diag.Diagnostics) {
	diags.AddAttributeError(
		path.Root(source.attribute),
		"Missing "+source.displayName,
		"Set the "+source.attribute+" provider attribute or the "+source.envVar+" environment variable.",
	)
}

func (p *KomodoProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAPIKeyResource,
	}
}

func (p *KomodoProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &KomodoProvider{
			version: version,
		}
	}
}
