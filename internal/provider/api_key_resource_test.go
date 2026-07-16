// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAPIKeyResource_Schema(t *testing.T) {
	resourceUnderTest := NewAPIKeyResource()
	var response resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &response)

	name, ok := response.Schema.Attributes["name"].(resourceschema.StringAttribute)
	if !ok || !name.Required || len(name.Validators) != 1 || !hasStringRequiresReplace(name.PlanModifiers) {
		t.Fatalf("unexpected name schema: %#v", name)
	}
	expires, ok := response.Schema.Attributes["expires"].(resourceschema.Int64Attribute)
	if !ok || !expires.Optional || !expires.Computed || len(expires.Validators) != 1 || !hasInt64RequiresReplace(expires.PlanModifiers) {
		t.Fatalf("unexpected expires schema: %#v", expires)
	}

	if _, ok := resourceUnderTest.(resource.ResourceWithImportState); !ok {
		t.Fatal("API key resource must support import")
	}
}

func TestAPIKeyResource_Validators(t *testing.T) {
	var nameResponse validator.StringResponse
	nonEmptyStringValidator{}.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("name"), ConfigValue: types.StringValue("")}, &nameResponse)
	if !nameResponse.Diagnostics.HasError() {
		t.Fatal("expected empty name validation error")
	}

	var expiresResponse validator.Int64Response
	int64AtLeastValidator{min: 0}.ValidateInt64(context.Background(), validator.Int64Request{Path: path.Root("expires"), ConfigValue: types.Int64Value(-1)}, &expiresResponse)
	if !expiresResponse.Diagnostics.HasError() {
		t.Fatal("expected negative expiration validation error")
	}
}

func hasStringRequiresReplace(modifiers []planmodifier.String) bool {
	return len(modifiers) == 1 && reflect.TypeOf(modifiers[0]) == reflect.TypeOf(stringplanmodifier.RequiresReplace())
}

func hasInt64RequiresReplace(modifiers []planmodifier.Int64) bool {
	return len(modifiers) == 1 && reflect.TypeOf(modifiers[0]) == reflect.TypeOf(int64planmodifier.RequiresReplace())
}
