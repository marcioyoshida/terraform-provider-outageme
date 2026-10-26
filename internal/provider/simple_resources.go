package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Channels, windows and domains have no update endpoint: any change replaces them.
var (
	replaceStr  = []planmodifier.String{stringplanmodifier.RequiresReplace()}
	replaceInt  = []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	replaceBool = []planmodifier.Bool{boolplanmodifier.RequiresReplace()}
	replaceList = []planmodifier.List{listplanmodifier.RequiresReplace()}
	keepID      = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
)

// findIn lists a collection and returns the item whose key equals id (nil: gone).
func findIn(ctx context.Context, c *Client, coll, key, id string) (map[string]any, error) {
	var out map[string][]map[string]any
	if err := c.do(ctx, "GET", "/v1/monitor/"+coll, nil, &out); err != nil {
		return nil, err
	}
	for _, it := range out[coll] {
		if fmt.Sprint(it[key]) == id {
			return it, nil
		}
	}
	return nil, nil
}

// ---- outageme_channel ---------------------------------------------------------------

func NewChannelResource() resource.Resource { return &channelResource{} }

type channelResource struct{ c *Client }

type channelModel struct {
	ID            types.String `tfsdk:"id"`
	Kind          types.String `tfsdk:"kind"`
	Target        types.String `tfsdk:"target"`
	Label         types.String `tfsdk:"label"`
	EU            types.Bool   `tfsdk:"eu"`
	Host          types.String `tfsdk:"host"`
	SigningSecret types.String `tfsdk:"signing_secret"`
}

func (r *channelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel"
}

func (r *channelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An alert channel. The API never returns targets or keys, so changing one replaces the channel.",
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Computed: true, PlanModifiers: keepID},
			"kind":   schema.StringAttribute{Required: true, PlanModifiers: replaceStr, Description: "email, slack, teams, webhook, pagerduty or opsgenie."},
			"target": schema.StringAttribute{Required: true, Sensitive: true, PlanModifiers: replaceStr, Description: "Webhook URL, PagerDuty routing key, Opsgenie API key, or for email the member's user id (the org id for the owner)."},
			"label":  schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: append(replaceStr, stringplanmodifier.UseStateForUnknown())},
			"eu":     schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: append(replaceBool, boolplanmodifier.UseStateForUnknown()), Description: "Opsgenie EU instance."},
			"host":   schema.StringAttribute{Computed: true, PlanModifiers: keepID},
			"signing_secret": schema.StringAttribute{Computed: true, Sensitive: true, PlanModifiers: keepID,
				Description: "webhook channels: the HMAC secret, returned once at creation."},
		},
	}
}

func (r *channelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.c = configureClient(req, resp)
}

func (r *channelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m channelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	b := map[string]any{"kind": m.Kind.ValueString(), "target": m.Target.ValueString()}
	putStr(b, "label", m.Label)
	putBool(b, "eu", m.EU)
	var out struct {
		Channel       map[string]any `json:"channel"`
		SigningSecret string         `json:"signing_secret"`
	}
	if err := r.c.do(ctx, "POST", "/v1/monitor/channels", b, &out); err != nil {
		resp.Diagnostics.AddError("Creating the channel failed", err.Error())
		return
	}
	m.ID, m.Label, m.Host = jStr(out.Channel, "channel_id"), jStr(out.Channel, "label"), jStr(out.Channel, "host")
	if m.Host.IsNull() {
		m.Host = types.StringValue("")
	}
	m.EU = types.BoolValue(m.EU.ValueBool())
	m.SigningSecret = types.StringValue(out.SigningSecret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *channelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m channelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	it, err := findIn(ctx, r.c, "channels", "channel_id", m.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading the channel failed", err.Error())
		return
	}
	if it == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	m.Kind, m.Label = jStr(it, "kind"), jStr(it, "label")
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *channelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Channels can't be updated", "Every attribute forces replacement; this is a provider bug.")
}

func (r *channelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m channelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if err := r.c.do(ctx, "DELETE", "/v1/monitor/channels/"+m.ID.ValueString(), nil, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting the channel failed", err.Error())
	}
}

// ---- outageme_maintenance_window -----------------------------------------------------

func NewWindowResource() resource.Resource { return &windowResource{} }

type windowResource struct{ c *Client }

type windowModel struct {
	ID      types.String `tfsdk:"id"`
	Label   types.String `tfsdk:"label"`
	Start   types.Int64  `tfsdk:"start"`
	End     types.Int64  `tfsdk:"end"`
	Days    types.List   `tfsdk:"days"`
	At      types.String `tfsdk:"at"`
	Minutes types.Int64  `tfsdk:"minutes"`
	Checks  types.List   `tfsdk:"checks"`
	Tags    types.List   `tfsdk:"tags"`
}

func (r *windowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_window"
}

func (r *windowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A maintenance window: alerts are held, results and incidents still recorded. Either start/end (one-off, Unix seconds) or days/at/minutes (weekly, UTC). Empty checks and tags = all checks.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true, PlanModifiers: keepID},
			"label":   schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: append(replaceStr, stringplanmodifier.UseStateForUnknown())},
			"start":   schema.Int64Attribute{Optional: true, PlanModifiers: replaceInt},
			"end":     schema.Int64Attribute{Optional: true, PlanModifiers: replaceInt},
			"days":    schema.ListAttribute{ElementType: types.Int64Type, Optional: true, PlanModifiers: replaceList, Description: "0 = Monday … 6 = Sunday."},
			"at":      schema.StringAttribute{Optional: true, PlanModifiers: replaceStr, Description: "HH:MM, UTC."},
			"minutes": schema.Int64Attribute{Optional: true, PlanModifiers: replaceInt},
			"checks":  schema.ListAttribute{ElementType: types.StringType, Optional: true, PlanModifiers: replaceList},
			"tags":    schema.ListAttribute{ElementType: types.StringType, Optional: true, PlanModifiers: replaceList},
		},
	}
}

func (r *windowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.c = configureClient(req, resp)
}

func (r *windowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m windowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	b := map[string]any{}
	putStr(b, "label", m.Label)
	putInt(b, "start", m.Start)
	putInt(b, "end", m.End)
	putStr(b, "at", m.At)
	putInt(b, "minutes", m.Minutes)
	putList[int64](ctx, b, "days", m.Days, &resp.Diagnostics)
	putList[string](ctx, b, "checks", m.Checks, &resp.Diagnostics)
	putList[string](ctx, b, "tags", m.Tags, &resp.Diagnostics)
	var out struct {
		Window map[string]any `json:"window"`
	}
	if err := r.c.do(ctx, "POST", "/v1/monitor/windows", b, &out); err != nil {
		resp.Diagnostics.AddError("Creating the window failed", err.Error())
		return
	}
	m.ID, m.Label = jStr(out.Window, "window_id"), jStr(out.Window, "label")
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *windowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m windowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	it, err := findIn(ctx, r.c, "windows", "window_id", m.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading the window failed", err.Error())
		return
	}
	if it == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	m.Label = jStr(it, "label")
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *windowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Windows can't be updated", "Every attribute forces replacement; this is a provider bug.")
}

func (r *windowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m windowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if err := r.c.do(ctx, "DELETE", "/v1/monitor/windows/"+m.ID.ValueString(), nil, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting the window failed", err.Error())
	}
}

// ---- outageme_domain -----------------------------------------------------------------

func NewDomainResource() resource.Resource { return &domainResource{} }

type domainResource struct{ c *Client }

type domainModel struct {
	ID       types.String `tfsdk:"id"`
	Host     types.String `tfsdk:"host"`
	Status   types.String `tfsdk:"status"`
	TXTName  types.String `tfsdk:"txt_name"`
	TXTValue types.String `tfsdk:"txt_value"`
	FileURL  types.String `tfsdk:"file_url"`
	FileBody types.String `tfsdk:"file_body"`
}

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A domain whose ownership unlocks every check type on it and its subdomains. Publish txt_value as a TXT record on txt_name (or serve file_body at file_url); outage.me re-checks automatically, or POST /v1/monitor/domains/{host}/verify.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true, PlanModifiers: keepID},
			"host":      schema.StringAttribute{Required: true, PlanModifiers: replaceStr},
			"status":    schema.StringAttribute{Computed: true, Description: "pending or verified."},
			"txt_name":  schema.StringAttribute{Computed: true, PlanModifiers: keepID},
			"txt_value": schema.StringAttribute{Computed: true, PlanModifiers: keepID},
			"file_url":  schema.StringAttribute{Computed: true, PlanModifiers: keepID},
			"file_body": schema.StringAttribute{Computed: true, PlanModifiers: keepID},
		},
	}
}

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.c = configureClient(req, resp)
}

func (r *domainResource) apply(m *domainModel, it map[string]any) {
	m.ID, m.Host, m.Status = jStr(it, "host"), jStr(it, "host"), jStr(it, "status")
	if how, ok := it["how"].(map[string]any); ok {
		m.TXTName, m.TXTValue = jStr(how, "txt_name"), jStr(how, "txt_value")
		m.FileURL, m.FileBody = jStr(how, "file_url"), jStr(how, "file_body")
	}
	for _, v := range []*types.String{&m.TXTName, &m.TXTValue, &m.FileURL, &m.FileBody} {
		if v.IsUnknown() {
			*v = types.StringValue("")
		}
	}
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m domainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	var out struct {
		Domain map[string]any `json:"domain"`
	}
	if err := r.c.do(ctx, "POST", "/v1/monitor/domains", map[string]any{"host": m.Host.ValueString()}, &out); err != nil {
		resp.Diagnostics.AddError("Adding the domain failed", err.Error())
		return
	}
	r.apply(&m, out.Domain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m domainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	it, err := findIn(ctx, r.c, "domains", "host", m.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading the domain failed", err.Error())
		return
	}
	if it == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	m.Status = jStr(it, "status")
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Domains can't be updated", "host forces replacement; this is a provider bug.")
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m domainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if err := r.c.do(ctx, "DELETE", "/v1/monitor/domains/"+m.ID.ValueString(), nil, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Removing the domain failed", err.Error())
	}
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("host"), req.ID)...)
}

// ---- outageme_status_page -------------------------------------------------------------

func NewPageResource() resource.Resource { return &pageResource{} }

type pageResource struct{ c *Client }

type pageModel struct {
	ID          types.String `tfsdk:"id"`
	Slug        types.String `tfsdk:"slug"`
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	Published   types.Bool   `tfsdk:"published"`
	Checks      types.List   `tfsdk:"checks"`
	URL         types.String `tfsdk:"url"`
}

func (r *pageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page"
}

func (r *pageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The organization's public status page at https://outage.me/s/<slug>/ (one per organization). Destroying it unpublishes the page.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, PlanModifiers: keepID},
			"slug":        schema.StringAttribute{Required: true, Description: "3-40 lowercase letters, digits and single hyphens."},
			"title":       schema.StringAttribute{Required: true},
			"description": schema.StringAttribute{Optional: true},
			"published":   schema.BoolAttribute{Optional: true, Computed: true, Description: "Default false."},
			"checks":      schema.ListAttribute{ElementType: types.StringType, Required: true, Description: "outageme_check ids to show."},
			"url":         schema.StringAttribute{Computed: true},
		},
	}
}

func (r *pageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.c = configureClient(req, resp)
}

func (r *pageResource) put(ctx context.Context, m *pageModel, published bool, d *diagHolder) (map[string]any, error) {
	b := map[string]any{"slug": m.Slug.ValueString(), "title": m.Title.ValueString(), "published": published}
	putStr(b, "description", m.Description)
	putList[string](ctx, b, "checks", m.Checks, d.d)
	var out struct {
		Page map[string]any `json:"page"`
	}
	err := r.c.do(ctx, "PUT", "/v1/monitor/page", b, &out)
	return out.Page, err
}

func (r *pageResource) set(m *pageModel, p map[string]any) {
	m.ID, m.Slug, m.Title = jStr(p, "slug"), jStr(p, "slug"), jStr(p, "title")
	m.Description, m.Published, m.Checks, m.URL = jStr(p, "description"), jBool(p, "published"), jStrList(p, "checks"), jStr(p, "url")
}

func (r *pageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m pageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	p, err := r.put(ctx, &m, m.Published.ValueBool(), &diagHolder{&resp.Diagnostics})
	if err != nil {
		resp.Diagnostics.AddError("Saving the status page failed", err.Error())
		return
	}
	r.set(&m, p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *pageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m pageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	var out struct {
		Page map[string]any `json:"page"`
	}
	if err := r.c.do(ctx, "GET", "/v1/monitor/page", nil, &out); err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Reading the status page failed", err.Error())
		return
	}
	r.set(&m, out.Page)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *pageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m pageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	p, err := r.put(ctx, &m, m.Published.ValueBool(), &diagHolder{&resp.Diagnostics})
	if err != nil {
		resp.Diagnostics.AddError("Saving the status page failed", err.Error())
		return
	}
	r.set(&m, p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *pageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m pageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if _, err := r.put(ctx, &m, false, &diagHolder{&resp.Diagnostics}); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Unpublishing the status page failed", err.Error())
	}
}
