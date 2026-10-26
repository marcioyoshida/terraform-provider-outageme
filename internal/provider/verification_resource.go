package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// outageme_domain_verification waits until the domain's proof is found, like
// aws_acm_certificate_validation: create the TXT record (any DNS provider) or the
// well-known file between outageme_domain and this resource, and make checks that need a
// verified domain depend on it.

func NewVerificationResource() resource.Resource { return &verificationResource{} }

type verificationResource struct{ c *Client }

type verificationModel struct {
	ID          types.String `tfsdk:"id"`
	Host        types.String `tfsdk:"host"`
	TimeoutSecs types.Int64  `tfsdk:"timeout_seconds"`
	Method      types.String `tfsdk:"method"`
}

func (r *verificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_verification"
}

func (r *verificationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Waits until an outageme_domain is verified (its TXT record or well-known file is found). Retries every 15 s; DNS can take minutes.",
		Attributes: map[string]schema.Attribute{
			"id":              schema.StringAttribute{Computed: true, PlanModifiers: keepID},
			"host":            schema.StringAttribute{Required: true, PlanModifiers: replaceStr},
			"timeout_seconds": schema.Int64Attribute{Optional: true, Description: "Default 600."},
			"method":          schema.StringAttribute{Computed: true, PlanModifiers: keepID, Description: "txt or file."},
		},
	}
}

func (r *verificationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.c = configureClient(req, resp)
}

func (r *verificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m verificationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	timeout := int64(600)
	if !m.TimeoutSecs.IsNull() {
		timeout = m.TimeoutSecs.ValueInt64()
	}
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	host := m.Host.ValueString()
	for {
		var out struct {
			Domain map[string]any `json:"domain"`
		}
		err := r.c.do(ctx, "POST", "/v1/monitor/domains/"+host+"/verify", nil, &out)
		if err == nil {
			m.ID, m.Method = types.StringValue(host), jStr(out.Domain, "method")
			if m.Method.IsNull() {
				m.Method = types.StringValue("")
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
			return
		}
		if e, ok := err.(*APIError); !ok || e.Status != 409 || time.Now().After(deadline) {
			resp.Diagnostics.AddError("Domain not verified", err.Error())
			return
		}
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Domain not verified", ctx.Err().Error())
			return
		case <-time.After(15 * time.Second):
		}
	}
}

func (r *verificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m verificationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	it, err := findIn(ctx, r.c, "domains", "host", m.Host.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading the domain failed", err.Error())
		return
	}
	if it == nil || jStr(it, "status").ValueString() != "verified" {
		resp.State.RemoveResource(ctx) // lapsed or removed: the next apply re-verifies
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *verificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m verificationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *verificationResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}
