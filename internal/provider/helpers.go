package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// providerDataFrom converts the opaque provider data passed to Configure on a
// resource or data source. It returns nil (without error) before the provider
// itself has been configured, which happens during validation.
func providerDataFrom(raw any, diags *diag.Diagnostics) *providerData {
	if raw == nil {
		return nil
	}
	data, ok := raw.(*providerData)
	if !ok {
		diags.AddError("Unexpected provider data type",
			fmt.Sprintf("Expected *providerData, got %T. This is a bug in the provider.", raw))
		return nil
	}
	return data
}

// --- Terraform -> API ---------------------------------------------------

func stringValue(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

func boolPointer(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return v.ValueBoolPointer()
}

func int64Pointer(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return v.ValueInt64Pointer()
}

// stringSlice returns nil for null/unknown lists so they are omitted from the
// API request; a known empty list returns a non-nil empty slice.
func stringSlice(ctx context.Context, v types.List, diags *diag.Diagnostics) *[]string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	out := []string{}
	diags.Append(v.ElementsAs(ctx, &out, false)...)
	return &out
}

func stringMap(ctx context.Context, v types.Map, diags *diag.Diagnostics) map[string]string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	out := map[string]string{}
	diags.Append(v.ElementsAs(ctx, &out, false)...)
	return out
}

// --- API -> Terraform ---------------------------------------------------

// stringOrNull maps the API's "unset" empty string to a null value.
func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func stringList(values []string) types.List {
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, elems)
}

func stringListPtr(values *[]string) types.List {
	if values == nil {
		return stringList(nil)
	}
	return stringList(*values)
}
