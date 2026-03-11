package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AccessDataSource{}
var _ datasource.DataSourceWithConfigure = &AccessDataSource{}

type AccessDataSource struct {
	client *NtfyClient
}

type AccessDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Username   types.String `tfsdk:"username"`
	Topic      types.String `tfsdk:"topic"`
	Permission types.String `tfsdk:"permission"`
}

func NewAccessDataSource() datasource.DataSource {
	return &AccessDataSource{}
}

func (d *AccessDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access"
}

func (d *AccessDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing ntfy access grant.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier (format: username/topic).",
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The username to look up.",
			},
			"topic": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The topic to look up.",
			},
			"permission": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The permission level: read-write, read-only, write-only, or deny.",
			},
		},
	}
}

func (d *AccessDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*NtfyClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected DataSource Configure Type", "Expected *NtfyClient")
		return
	}
	d.client = client
}

func (d *AccessDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AccessDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, statusCode, err := d.client.AdminRequest(ctx, http.MethodGet, "/v1/users", nil)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to list users: %s", err))
		return
	}

	if statusCode != http.StatusOK {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unexpected status %d listing users", statusCode))
		return
	}

	var users []apiUserResponse
	if err := json.Unmarshal(respBody, &users); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Failed to parse users: %s", err))
		return
	}

	for _, u := range users {
		if u.Username == config.Username.ValueString() {
			for _, g := range u.Grants {
				if g.Topic == config.Topic.ValueString() {
					config.ID = types.StringValue(config.Username.ValueString() + "/" + config.Topic.ValueString())
					config.Permission = types.StringValue(g.Permission)
					resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
					return
				}
			}
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Access grant for %s on topic %s not found", config.Username.ValueString(), config.Topic.ValueString()))
			return
		}
	}

	resp.Diagnostics.AddError("Not Found", fmt.Sprintf("User %s not found", config.Username.ValueString()))
}
