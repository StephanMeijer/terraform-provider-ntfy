package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &NtfyProvider{}

type NtfyProvider struct{}

type NtfyProviderModel struct {
	URL      types.String `tfsdk:"url"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func New() provider.Provider {
	return &NtfyProvider{}
}

func (p *NtfyProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ntfy"
}

func (p *NtfyProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Optional:            true,
				Description:         "The URL of the ntfy server. Defaults to http://localhost:80. Can be set via NTFY_URL env var.",
				MarkdownDescription: "The URL of the ntfy server. Defaults to `http://localhost:80`. Can be set via `NTFY_URL` environment variable.",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				Description:         "Admin username for the ntfy server. Defaults to admin. Can be set via NTFY_USERNAME env var.",
				MarkdownDescription: "Admin username for the ntfy server. Defaults to `admin`. Can be set via `NTFY_USERNAME` environment variable.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				Description:         "Admin password for the ntfy server. Can be set via NTFY_PASSWORD env var.",
				MarkdownDescription: "Admin password for the ntfy server. Can be set via `NTFY_PASSWORD` environment variable.",
			},
		},
	}
}

func (p *NtfyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config NtfyProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := "http://localhost:80"
	username := "admin"
	password := ""

	if v, ok := os.LookupEnv("NTFY_URL"); ok {
		url = v
	}
	if v, ok := os.LookupEnv("NTFY_USERNAME"); ok {
		username = v
	}
	if v, ok := os.LookupEnv("NTFY_PASSWORD"); ok {
		password = v
	}

	if !config.URL.IsNull() {
		url = config.URL.ValueString()
	}
	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}
	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	client := NewNtfyClient(url, username, password)

	_, statusCode, err := client.AdminRequest(ctx, http.MethodGet, "/v1/health", nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cannot reach ntfy server",
			fmt.Sprintf("Cannot reach ntfy server at %s. Is it running? Error: %v", url, err),
		)
		return
	}

	if statusCode == http.StatusUnauthorized {
		resp.Diagnostics.AddError(
			"Authentication failed",
			"Authentication failed. Check username and password.",
		)
		return
	}

	if statusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to connect to ntfy server",
			fmt.Sprintf("Failed to connect to ntfy server at %s: HTTP %d", url, statusCode),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *NtfyProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
		NewAccessResource,
		NewTokenResource,
	}
}

func (p *NtfyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserDataSource,
		NewAccessDataSource,
	}
}
