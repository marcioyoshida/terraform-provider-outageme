package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewCheckResource() resource.Resource { return &checkResource{} }

type checkResource struct{ c *Client }

type checkModel struct {
	ID            types.String  `tfsdk:"id"`
	Type          types.String  `tfsdk:"type"`
	Name          types.String  `tfsdk:"name"`
	URL           types.String  `tfsdk:"url"`
	Method        types.String  `tfsdk:"method"`
	Headers       types.Map     `tfsdk:"headers"`
	Body          types.String  `tfsdk:"body"`
	Expect        types.List    `tfsdk:"expect"`
	Keyword       types.String  `tfsdk:"keyword"`
	KeywordAbsent types.Bool    `tfsdk:"keyword_absent"`
	MaxMs         types.Int64   `tfsdk:"max_ms"`
	Follow        types.Bool    `tfsdk:"follow"`
	Host          types.String  `tfsdk:"host"`
	Port          types.Int64   `tfsdk:"port"`
	Banner        types.String  `tfsdk:"banner"`
	Record        types.String  `tfsdk:"record"`
	Answer        types.String  `tfsdk:"answer"`
	Interval      types.Int64   `tfsdk:"interval"`
	Timeout       types.Int64   `tfsdk:"timeout"`
	Locations     types.List    `tfsdk:"locations"`
	Tags          types.List    `tfsdk:"tags"`
	Deps          types.List    `tfsdk:"deps"`
	Paused        types.Bool    `tfsdk:"paused"`
	SLA           types.Float64 `tfsdk:"sla"`
	AlertAfter    types.Int64   `tfsdk:"alert_after"`
	RealertMin    types.Int64   `tfsdk:"realert_min"`
	EscalateMin   types.Int64   `tfsdk:"escalate_min"`
	Channels      types.List    `tfsdk:"channels"`
	EscalateTo    types.List    `tfsdk:"escalate_to"`
	Verified      types.Bool    `tfsdk:"verified"`
}

func (r *checkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check"
}

func (r *checkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Description: desc}
	}
	strC := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Description: desc}
	}
	intC := func(desc string) schema.Int64Attribute {
		return schema.Int64Attribute{Optional: true, Computed: true, Description: desc}
	}
	boolC := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Description: desc}
	}
	strList := func(desc string) schema.ListAttribute {
		return schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: desc}
	}
	resp.Schema = schema.Schema{
		Description: "An uptime check on your own endpoint, run from 3-5 locations. Until its domain is verified (outageme_domain), a check is limited to a plain GET on port 80/443 every 5 minutes or slower.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"type":           strC("http (default), tcp or dns."),
			"name":           strC("Display name; defaults from the target."),
			"url":            str("http checks: the http(s) URL."),
			"method":         strC("http checks: GET (default), HEAD, POST, PUT or OPTIONS."),
			"headers":        schema.MapAttribute{ElementType: types.StringType, Optional: true, Sensitive: true, Description: "http checks: request headers (verified domains only)."},
			"body":           str("http checks: request body for POST/PUT."),
			"expect":         schema.ListAttribute{ElementType: types.Int64Type, Optional: true, Description: "Accepted HTTP statuses; omitted = 200-399."},
			"keyword":        str("http checks: the page must contain this text."),
			"keyword_absent": boolC("Invert keyword: the page must NOT contain it."),
			"max_ms":         intC("Fail when slower than this many milliseconds (0 = off)."),
			"follow":         boolC("Follow redirects."),
			"host":           strC("tcp and dns checks: the host name (http: derived from url)."),
			"port":           intC("tcp checks: the port (http: derived from url)."),
			"banner":         str("tcp checks: the greeting must contain this text."),
			"record":         strC("dns checks: A, AAAA, CNAME, MX, TXT or NS."),
			"answer":         str("dns checks: the expected answer."),
			"interval":       intC("Minutes between checks: 1, 5 (default) or 15."),
			"timeout":        intC("Seconds, 1-30 (default 10)."),
			"locations": schema.ListAttribute{ElementType: types.StringType, Optional: true, Computed: true,
				Description: "At least 3 of us-east-1, eu-west-2, eu-central-1, ap-south-1, ap-southeast-1."},
			"tags":         strList("Short labels, for maintenance windows and filters."),
			"deps":         strList("outage.me provider names this endpoint depends on (shown on incidents)."),
			"paused":       boolC("Pause the check (closes an open incident quietly)."),
			"sla":          schema.Float64Attribute{Optional: true, Description: "SLA target %, 90-99.999."},
			"alert_after":  intC("Alert after this many confirmed-down minutes (default 1)."),
			"realert_min":  intC("Repeat the alert every N minutes while down (0 = never)."),
			"escalate_min": intC("Escalate if unacknowledged after N minutes (0 = never)."),
			"channels":     strList("outageme_channel ids to alert."),
			"escalate_to":  strList("outageme_channel ids to escalate to."),
			"verified":     schema.BoolAttribute{Computed: true, Description: "Whether a verified domain covers the host."},
		},
	}
}

func (r *checkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.c = configureClient(req, resp)
}

func (r *checkResource) body(ctx context.Context, m *checkModel, resp *diagHolder) map[string]any {
	b := map[string]any{}
	for k, v := range map[string]types.String{"type": m.Type, "name": m.Name, "url": m.URL, "method": m.Method,
		"body": m.Body, "keyword": m.Keyword, "host": m.Host, "banner": m.Banner, "record": m.Record, "answer": m.Answer} {
		putStr(b, k, v)
	}
	for k, v := range map[string]types.Int64{"max_ms": m.MaxMs, "port": m.Port, "interval": m.Interval, "timeout": m.Timeout} {
		putInt(b, k, v)
	}
	for k, v := range map[string]types.Bool{"keyword_absent": m.KeywordAbsent, "follow": m.Follow, "paused": m.Paused} {
		putBool(b, k, v)
	}
	putMap(ctx, b, "headers", m.Headers, resp.d)
	putList[int64](ctx, b, "expect", m.Expect, resp.d)
	putList[string](ctx, b, "locations", m.Locations, resp.d)
	putList[string](ctx, b, "tags", m.Tags, resp.d)
	putList[string](ctx, b, "deps", m.Deps, resp.d)
	// rule fields are always sent, so removing one from the config clears it
	b["sla"] = nil
	putFloat(b, "sla", m.SLA)
	for k, v := range map[string]types.Int64{"alert_after": m.AlertAfter, "realert_min": m.RealertMin, "escalate_min": m.EscalateMin} {
		putInt(b, k, v)
	}
	b["channels"], b["escalate_to"] = []string{}, []string{}
	putList[string](ctx, b, "channels", m.Channels, resp.d)
	putList[string](ctx, b, "escalate_to", m.EscalateTo, resp.d)
	return b
}

func withDefault(v types.Int64, d int64) types.Int64 {
	if v.IsNull() {
		return types.Int64Value(d)
	}
	return v
}

func (r *checkResource) fill(m *checkModel, c map[string]any) {
	m.ID = jStr(c, "check_id")
	m.Type, m.Name, m.URL = jStr(c, "type"), jStr(c, "name"), jStr(c, "url")
	m.Method, m.Body, m.Keyword = jStr(c, "method"), jStr(c, "body"), jStr(c, "keyword")
	m.Host, m.Banner, m.Record, m.Answer = jStr(c, "host"), jStr(c, "banner"), jStr(c, "record"), jStr(c, "answer")
	m.KeywordAbsent, m.Follow, m.Paused = jBool(c, "keyword_absent"), jBool(c, "follow"), jBool(c, "paused")
	m.MaxMs, m.Port, m.Interval, m.Timeout = withDefault(jInt(c, "max_ms"), 0), jInt(c, "port"), jInt(c, "interval"), jInt(c, "timeout")
	m.Headers, m.Expect = jStrMap(c, "headers"), jIntList(c, "expect")
	m.Locations, m.Tags, m.Deps = jStrList(c, "locations"), jStrList(c, "tags"), jStrList(c, "deps")
	m.SLA = jFloat(c, "sla")
	m.AlertAfter = withDefault(jInt(c, "alert_after"), 1)
	m.RealertMin, m.EscalateMin = withDefault(jInt(c, "realert_min"), 0), withDefault(jInt(c, "escalate_min"), 0)
	m.Channels, m.EscalateTo = jStrList(c, "channels"), jStrList(c, "escalate_to")
	m.Verified = jBool(c, "verified")
}

type checkEnvelope struct {
	Check map[string]any `json:"check"`
}

func (r *checkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m checkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var out checkEnvelope
	if err := r.c.do(ctx, "POST", "/v1/monitor/checks", r.body(ctx, &m, &diagHolder{&resp.Diagnostics}), &out); err != nil {
		resp.Diagnostics.AddError("Creating the check failed", err.Error())
		return
	}
	r.fill(&m, out.Check)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *checkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m checkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var out checkEnvelope
	if err := r.c.do(ctx, "GET", "/v1/monitor/checks/"+m.ID.ValueString(), nil, &out); err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Reading the check failed", err.Error())
		return
	}
	r.fill(&m, out.Check)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *checkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m, st checkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &st)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var out checkEnvelope
	if err := r.c.do(ctx, "PUT", "/v1/monitor/checks/"+st.ID.ValueString(), r.body(ctx, &m, &diagHolder{&resp.Diagnostics}), &out); err != nil {
		resp.Diagnostics.AddError("Updating the check failed", err.Error())
		return
	}
	r.fill(&m, out.Check)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *checkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m checkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if err := r.c.do(ctx, "DELETE", "/v1/monitor/checks/"+m.ID.ValueString(), nil, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting the check failed", err.Error())
	}
}

func (r *checkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
