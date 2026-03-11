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

var _ datasource.DataSource = &UserDataSource{}
var _ datasource.DataSourceWithConfigure = &UserDataSource{}

type UserDataSource struct {
	client *NtfyClient
}

type UserDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	Role     types.String `tfsdk:"role"`
}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

func (d *UserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing ntfy user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier for this user (same as username).",
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The username to look up.",
			},
			"role": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The role of the user (user or admin).",
			},
		},
	}
}

func (d *UserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config UserDataSourceModel
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
			config.ID = config.Username
			config.Role = types.StringValue(u.Role)
			resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
			return
		}
	}

	resp.Diagnostics.AddError("Not Found", fmt.Sprintf("User %s not found", config.Username.ValueString()))
}
