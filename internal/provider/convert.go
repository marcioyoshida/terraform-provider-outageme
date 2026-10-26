package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The API returns JSON numbers as float64; empty strings and lists mean "not set", and are
// mapped back to null so a config that omits them never shows a diff.

func jStr(m map[string]any, k string) types.String {
	if v, ok := m[k].(string); ok && v != "" {
		return types.StringValue(v)
	}
	return types.StringNull()
}

func jInt(m map[string]any, k string) types.Int64 {
	if v, ok := m[k].(float64); ok {
		return types.Int64Value(int64(v))
	}
	return types.Int64Null()
}

func jFloat(m map[string]any, k string) types.Float64 {
	if v, ok := m[k].(float64); ok {
		return types.Float64Value(v)
	}
	return types.Float64Null()
}

func jBool(m map[string]any, k string) types.Bool {
	if v, ok := m[k].(bool); ok {
		return types.BoolValue(v)
	}
	return types.BoolValue(false)
}

func jStrList(m map[string]any, k string) types.List {
	raw, _ := m[k].([]any)
	if len(raw) == 0 {
		return types.ListNull(types.StringType)
	}
	vals := make([]attr.Value, 0, len(raw))
	for _, x := range raw {
		vals = append(vals, types.StringValue(fmt.Sprint(x)))
	}
	return types.ListValueMust(types.StringType, vals)
}

func jIntList(m map[string]any, k string) types.List {
	raw, _ := m[k].([]any)
	if len(raw) == 0 {
		return types.ListNull(types.Int64Type)
	}
	vals := make([]attr.Value, 0, len(raw))
	for _, x := range raw {
		if f, ok := x.(float64); ok {
			vals = append(vals, types.Int64Value(int64(f)))
		}
	}
	return types.ListValueMust(types.Int64Type, vals)
}

func jStrMap(m map[string]any, k string) types.Map {
	raw, _ := m[k].(map[string]any)
	if len(raw) == 0 {
		return types.MapNull(types.StringType)
	}
	vals := map[string]attr.Value{}
	for kk, v := range raw {
		vals[kk] = types.StringValue(fmt.Sprint(v))
	}
	return types.MapValueMust(types.StringType, vals)
}

// put* add a planned value to an API body only when it is set.

func putStr(b map[string]any, k string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() {
		b[k] = v.ValueString()
	}
}

func putInt(b map[string]any, k string, v types.Int64) {
	if !v.IsNull() && !v.IsUnknown() {
		b[k] = v.ValueInt64()
	}
}

func putFloat(b map[string]any, k string, v types.Float64) {
	if !v.IsNull() && !v.IsUnknown() {
		b[k] = v.ValueFloat64()
	}
}

func putBool(b map[string]any, k string, v types.Bool) {
	if !v.IsNull() && !v.IsUnknown() {
		b[k] = v.ValueBool()
	}
}

func putList[T any](ctx context.Context, b map[string]any, k string, v types.List, d *diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	var out []T
	d.Append(v.ElementsAs(ctx, &out, false)...)
	b[k] = out
}

func putMap(ctx context.Context, b map[string]any, k string, v types.Map, d *diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	out := map[string]string{}
	d.Append(v.ElementsAs(ctx, &out, false)...)
	b[k] = out
}

func clientFrom(data any, d *diag.Diagnostics) *Client {
	if data == nil {
		return nil
	}
	c, ok := data.(*Client)
	if !ok {
		d.AddError("Provider not configured", fmt.Sprintf("unexpected provider data %T", data))
	}
	return c
}

func configureClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *Client {
	return clientFrom(req.ProviderData, &resp.Diagnostics)
}

// diagHolder lets helpers append to whichever response's diagnostics.
type diagHolder struct{ d *diag.Diagnostics }
