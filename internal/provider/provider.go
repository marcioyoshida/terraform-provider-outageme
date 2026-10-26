package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type outagemeProvider struct{ version string }

type providerModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
	Org     types.String `tfsdk:"org"`
}

// New returns the provider factory providerserver.Serve expects.
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &outagemeProvider{version: version} }
}

func (p *outagemeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "outageme"
	resp.Version = p.version
}

func (p *outagemeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "outage.me Monitor: uptime checks on your own endpoints from up to 5 locations.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional: true, Sensitive: true,
				Description: "An outage.me API key (Keys tab of the Pro app). Defaults to OUTAGEME_API_KEY.",
			},
			"base_url": schema.StringAttribute{
				Optional: true, Description: "Defaults to https://outage.me (or OUTAGEME_BASE_URL).",
			},
			"org": schema.StringAttribute{
				Optional:    true,
				Description: "Manage another organization you are a member of, at your role there. Defaults to the key's own organization.",
			},
		},
	}
}

func (p *outagemeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	key := cfg.APIKey.ValueString()
	if key == "" {
		key = os.Getenv("OUTAGEME_API_KEY")
	}
	if key == "" {
		resp.Diagnostics.AddError("Missing API key", "Set api_key in the provider block or OUTAGEME_API_KEY.")
		return
	}
	base := cfg.BaseURL.ValueString()
	if base == "" {
		base = os.Getenv("OUTAGEME_BASE_URL")
	}
	if base == "" {
		base = "https://outage.me"
	}
	c := &Client{BaseURL: base, APIKey: key, Org: cfg.Org.ValueString(),
		UA: "terraform-provider-outageme/" + p.version}
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *outagemeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCheckResource, NewChannelResource, NewWindowResource, NewDomainResource, NewVerificationResource, NewPageResource,
	}
}

func (p *outagemeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
