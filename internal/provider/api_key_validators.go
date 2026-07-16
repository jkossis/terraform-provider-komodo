// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type nonEmptyStringValidator struct {
	description string
}

func (v nonEmptyStringValidator) Description(context.Context) string {
	return v.description
}

func (v nonEmptyStringValidator) MarkdownDescription(context.Context) string {
	return v.description
}

func (v nonEmptyStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() && req.ConfigValue.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid API Key Name", "API key name must not be empty.")
	}
}

type int64AtLeastValidator struct {
	min         int64
	description string
}

func (v int64AtLeastValidator) Description(context.Context) string {
	return v.description
}

func (v int64AtLeastValidator) MarkdownDescription(context.Context) string {
	return v.description
}

func (v int64AtLeastValidator) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() && req.ConfigValue.ValueInt64() < v.min {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid API Key Expiration", fmt.Sprintf("API key expiration must be greater than or equal to %d.", v.min))
	}
}
