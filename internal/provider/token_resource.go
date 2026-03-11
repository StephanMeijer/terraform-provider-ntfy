package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = &TokenResource{}
	_ resource.ResourceWithConfigure = &TokenResource{}
)

type TokenResource struct {
	client *NtfyClient
}

type TokenResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Label    types.String `tfsdk:"label"`
	Token    types.String `tfsdk:"token"`
	Expires  types.Int64  `tfsdk:"expires"`
}

func NewTokenResource() resource.Resource {
	return &TokenResource{}
}

func (r *TokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

func (r *TokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages an ntfy API token. Uses user credentials (not admin) for authentication.",
		MarkdownDescription: "Manages an ntfy API token. Uses user credentials (not admin) for authentication. This resource does not support import.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier for this token (same as the token value). This resource does not support import.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "The username to create the token for.",
				MarkdownDescription: "The username to create the token for. Changing this forces a new token to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "The password of the user to create the token for.",
				MarkdownDescription: "The password of the user to create the token for. Changing this forces a new token to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"label": schema.StringAttribute{
				Optional:            true,
				Description:         "An optional label for the token.",
				MarkdownDescription: "An optional human-readable label for the token. Changing this forces a new token to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				Description:         "The generated API token.",
				MarkdownDescription: "The generated API token value. This is sensitive and will not be shown in plan output.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expires": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				Description:         "Token expiry as Unix timestamp. 0 means no expiry.",
				MarkdownDescription: "Token expiry as Unix timestamp (seconds since epoch). Use `0` for no expiry. Changing this forces a new token to be created.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *TokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*NtfyClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			"Expected *NtfyClient, got something else.")
		return
	}
	r.client = client
}

func (r *TokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	expires := plan.Expires.ValueInt64()

	var label *string
	if !plan.Label.IsNull() && !plan.Label.IsUnknown() {
		l := plan.Label.ValueString()
		label = &l
	}

	body := apiAccountTokenIssueRequest{
		Label:   label,
		Expires: &expires,
	}

	// Use USER Basic Auth, not admin
	respBody, statusCode, err := r.client.UserRequest(
		ctx,
		http.MethodPost,
		"/v1/account/token",
		body,
		plan.Username.ValueString(),
		plan.Password.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to create token: %s", err))
		return
	}

	if statusCode != http.StatusOK {
		switch statusCode {
		case http.StatusBadRequest:
			resp.Diagnostics.AddError("Bad Request", fmt.Sprintf("ntfy returned 400: %s", string(respBody)))
		case http.StatusUnauthorized:
			resp.Diagnostics.AddError("Unauthorized", "Invalid username or password")
		case http.StatusForbidden:
			resp.Diagnostics.AddError("Forbidden", "User does not have permission to create tokens")
		case http.StatusInternalServerError:
			resp.Diagnostics.AddError("Server Error", fmt.Sprintf("ntfy server error: %s", string(respBody)))
		default:
			resp.Diagnostics.AddError("Unexpected Error", fmt.Sprintf("HTTP %d: %s", statusCode, string(respBody)))
		}
		return
	}

	var tokenResp apiAccountTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Failed to parse token response: %s", err))
		return
	}

	plan.Token = types.StringValue(tokenResp.Token)
	plan.ID = types.StringValue(tokenResp.Token)
	plan.Expires = types.Int64Value(tokenResp.Expires)
	if tokenResp.Label != "" {
		plan.Label = types.StringValue(tokenResp.Label)
	}

	tflog.Debug(ctx, "Created ntfy token", map[string]interface{}{
		"username": plan.Username.ValueString(),
		"label":    plan.Label.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use Bearer auth with the token to check validity
	_, statusCode, err := r.client.TokenRequest(
		ctx,
		http.MethodGet,
		"/v1/account",
		state.Token.ValueString(),
		nil,
	)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to read account: %s", err))
		return
	}

	if statusCode == http.StatusUnauthorized {
		tflog.Debug(ctx, "Token is no longer valid, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	if statusCode != http.StatusOK {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unexpected status code %d reading account", statusCode))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update needed - ForceNew on all attributes
}

func (r *TokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// DELETE /v1/account/token with Bearer auth and X-Token header
	headers := map[string]string{
		"X-Token": state.Token.ValueString(),
	}

	respBody, statusCode, err := r.client.TokenRequest(
		ctx,
		http.MethodDelete,
		"/v1/account/token",
		state.Token.ValueString(),
		headers,
	)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to delete token: %s", err))
		return
	}

	if statusCode != http.StatusOK {
		switch statusCode {
		case http.StatusUnauthorized:
			resp.Diagnostics.AddError("Unauthorized", "Token is no longer valid or already deleted")
		case http.StatusForbidden:
			resp.Diagnostics.AddError("Forbidden", "User does not have permission to delete this token")
		case http.StatusNotFound:
			// Token already gone — treat as success
			tflog.Debug(ctx, "Token not found during delete, treating as already deleted")
		default:
			resp.Diagnostics.AddError("Unexpected Error", fmt.Sprintf("HTTP %d deleting token: %s", statusCode, string(respBody)))
		}
		return
	}

	tflog.Debug(ctx, "Deleted ntfy token", map[string]interface{}{
		"username": state.Username.ValueString(),
	})
}
