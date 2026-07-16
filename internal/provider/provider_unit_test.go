// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestKomodoProvider_Metadata_usesProviderTypeName(t *testing.T) {
	// Given
	provider := &KomodoProvider{version: "test"}
	response := &frameworkprovider.MetadataResponse{}

	// When
	provider.Metadata(context.Background(), frameworkprovider.MetadataRequest{}, response)

	// Then
	if response.TypeName != providerTypeName {
		t.Fatalf("expected provider type name %q, got %q", providerTypeName, response.TypeName)
	}
	if typeNamePrefix != providerTypeName {
		t.Fatalf("expected typeNamePrefix to match providerTypeName, got %q and %q", typeNamePrefix, providerTypeName)
	}
}

func TestKomodoProvider_Schema_keepsEnvBackedAttributesOptionalAndSensitive(t *testing.T) {
	// Given
	provider := &KomodoProvider{}
	response := &frameworkprovider.SchemaResponse{}

	// When
	provider.Schema(context.Background(), frameworkprovider.SchemaRequest{}, response)

	// Then
	assertStringAttribute(t, response.Schema.Attributes["endpoint"], attributeExpectation{
		optional: true,
		envVar:   komodoEndpointEnv,
	})
	assertStringAttribute(t, response.Schema.Attributes["username"], attributeExpectation{
		optional: true,
		envVar:   komodoUsernameEnv,
	})
	assertStringAttribute(t, response.Schema.Attributes["password"], attributeExpectation{
		optional:  true,
		sensitive: true,
		envVar:    komodoPasswordEnv,
	})
}

func TestProviderConfigFrom_prefersConfigOverEnvironment(t *testing.T) {
	// Given
	data := KomodoProviderModel{
		Endpoint: types.StringValue("https://config.example"),
		Username: types.StringValue("config-user"),
		Password: types.StringValue("config-password"),
	}

	// When
	config, diagnostics := providerConfigFrom(data, func(name string) string {
		return map[string]string{
			komodoEndpointEnv: "https://env.example",
			komodoUsernameEnv: "env-user",
			komodoPasswordEnv: "env-password",
		}[name]
	})

	// Then
	if diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got %v", diagnostics)
	}
	if config.endpoint != "https://config.example" {
		t.Fatalf("expected config endpoint, got %q", config.endpoint)
	}
	if config.username != "config-user" {
		t.Fatalf("expected config username, got %q", config.username)
	}
	if config.password != "config-password" {
		t.Fatalf("expected config password, got %q", config.password)
	}
}

func TestProviderConfigFrom_usesEnvironmentWhenConfigMissing(t *testing.T) {
	// Given
	data := KomodoProviderModel{
		Endpoint: types.StringNull(),
		Username: types.StringNull(),
		Password: types.StringNull(),
	}

	// When
	config, diagnostics := providerConfigFrom(data, func(name string) string {
		return map[string]string{
			komodoEndpointEnv: "https://env.example",
			komodoUsernameEnv: "env-user",
			komodoPasswordEnv: "env-password",
		}[name]
	})

	// Then
	if diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got %v", diagnostics)
	}
	if config.endpoint != "https://env.example" {
		t.Fatalf("expected environment endpoint, got %q", config.endpoint)
	}
	if config.username != "env-user" {
		t.Fatalf("expected environment username, got %q", config.username)
	}
	if config.password != "env-password" {
		t.Fatalf("expected environment password, got %q", config.password)
	}
}

func TestProviderConfigFrom_returnsAttributeDiagnosticsWhenCredentialsMissing(t *testing.T) {
	// Given
	data := KomodoProviderModel{
		Endpoint: types.StringNull(),
		Username: types.StringNull(),
		Password: types.StringNull(),
	}

	// When
	_, diagnostics := providerConfigFrom(data, func(string) string { return "" })

	// Then
	if !diagnostics.HasError() {
		t.Fatal("expected missing config diagnostics")
	}
	assertDiagnosticPaths(t, diagnostics, []string{"endpoint", "username", "password"})
}

func TestProviderConfigFrom_doesNotUseEnvironmentForUnknownValues(t *testing.T) {
	data := KomodoProviderModel{
		Endpoint: types.StringUnknown(),
		Username: types.StringUnknown(),
		Password: types.StringUnknown(),
	}

	config, diagnostics := providerConfigFrom(data, func(string) string { return "from-environment" })

	if !diagnostics.HasError() {
		t.Fatal("expected unknown config diagnostics")
	}
	if config.endpoint != "" || config.username != "" || config.password != "" {
		t.Fatalf("unknown values must not fall back to the environment: %#v", config)
	}
	assertDiagnosticPaths(t, diagnostics, []string{"endpoint", "username", "password"})
}

func TestProviderConfigFrom_rejectsEmptyExplicitValues(t *testing.T) {
	data := KomodoProviderModel{
		Endpoint: types.StringValue(""),
		Username: types.StringValue(""),
		Password: types.StringValue(""),
	}

	config, diagnostics := providerConfigFrom(data, func(string) string { return "from-environment" })

	if !diagnostics.HasError() {
		t.Fatal("expected empty config diagnostics")
	}
	if config.endpoint != "" || config.username != "" || config.password != "" {
		t.Fatalf("empty values must not fall back to the environment: %#v", config)
	}
}

func TestKomodoProvider_Resources_registerAPIKeyWithSharedPrefix(t *testing.T) {
	// Given
	provider := &KomodoProvider{}
	resources := provider.Resources(context.Background())

	// When
	metadataResponse := &resource.MetadataResponse{}
	resources[0]().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: typeNamePrefix}, metadataResponse)

	// Then
	if len(resources) != 1 {
		t.Fatalf("expected one resource, got %d", len(resources))
	}
	if metadataResponse.TypeName != "komodo_api_key" {
		t.Fatalf("expected komodo_api_key resource, got %q", metadataResponse.TypeName)
	}
}

func TestKomodoProvider_DataSources_areExplicitlyEmpty(t *testing.T) {
	// Given
	provider := &KomodoProvider{}

	// When
	dataSources := provider.DataSources(context.Background())

	// Then
	if len(dataSources) != 0 {
		t.Fatalf("expected no data sources, got %d", len(dataSources))
	}
}

type attributeExpectation struct {
	optional  bool
	sensitive bool
	envVar    string
}

func assertStringAttribute(t *testing.T, attribute providerschema.Attribute, expectation attributeExpectation) {
	t.Helper()

	stringAttribute, ok := attribute.(providerschema.StringAttribute)
	if !ok {
		t.Fatalf("expected string attribute, got %T", attribute)
	}
	if stringAttribute.Optional != expectation.optional {
		t.Fatalf("expected optional=%t, got %t", expectation.optional, stringAttribute.Optional)
	}
	if stringAttribute.Sensitive != expectation.sensitive {
		t.Fatalf("expected sensitive=%t, got %t", expectation.sensitive, stringAttribute.Sensitive)
	}
	if !strings.Contains(stringAttribute.MarkdownDescription, expectation.envVar) {
		t.Fatalf("expected description to mention %s, got %q", expectation.envVar, stringAttribute.MarkdownDescription)
	}
}

func assertDiagnosticPaths(t *testing.T, diagnostics diag.Diagnostics, want []string) {
	t.Helper()

	got := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		diagnosticWithPath, ok := diagnostic.(diag.DiagnosticWithPath)
		if !ok {
			t.Fatalf("expected attribute diagnostic, got %T", diagnostic)
		}
		got = append(got, diagnosticWithPath.Path().String())
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("expected diagnostic paths %v, got %v", want, got)
	}
}
