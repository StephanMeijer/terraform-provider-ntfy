package provider

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithConfigure   = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
)

type UserResource struct {
	client *NtfyClient
}

type UserResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Role     types.String `tfsdk:"role"`
	Tier     types.String `tfsdk:"tier"`
}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

func (r *UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an ntfy user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier for this user (same as username).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username for the ntfy user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 32),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9_.\-]+$`),
						"Username must contain only letters, digits, underscores, dots, or hyphens",
					),
				},
			},
			"password": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The auto-generated password for the ntfy user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"role": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The role of the user (user or admin).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tier": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The tier to assign to the user. Tiers control rate limits and storage quotas. Leave empty for the default tier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	password, err := generatePassword(32)
	if err != nil {
		resp.Diagnostics.AddError("Password Generation Error", err.Error())
		return
	}

	body := apiUserAddOrUpdateRequest{
		Username: plan.Username.ValueString(),
		Password: password,
		Tier:     plan.Tier.ValueString(),
	}

	respBody, statusCode, err := r.client.AdminRequest(ctx, http.MethodPost, "/v1/users", body)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to create user: %s", err))
		return
	}

	if statusCode != http.StatusOK {
		switch statusCode {
		case http.StatusBadRequest:
			resp.Diagnostics.AddError("Bad Request", fmt.Sprintf("ntfy returned 400: %s", string(respBody)))
		case http.StatusUnauthorized:
			resp.Diagnostics.AddError("Unauthorized", "Check provider credentials (username/password)")
		case http.StatusForbidden:
			resp.Diagnostics.AddError("Forbidden", "Admin privileges required")
		case http.StatusConflict:
			resp.Diagnostics.AddError("Conflict", fmt.Sprintf("User %s already exists", plan.Username.ValueString()))
		case http.StatusInternalServerError:
			resp.Diagnostics.AddError("Server Error", fmt.Sprintf("ntfy server error: %s", string(respBody)))
		default:
			resp.Diagnostics.AddError("Unexpected Error", fmt.Sprintf("HTTP %d: %s", statusCode, string(respBody)))
		}
		return
	}

	tflog.Debug(ctx, "Created ntfy user", map[string]interface{}{"username": plan.Username.ValueString()})

	plan.Password = types.StringValue(password)
	plan.ID = plan.Username
	plan.Role = types.StringValue("user")
	if plan.Tier.IsUnknown() {
		plan.Tier = types.StringValue("")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, statusCode, err := r.client.AdminRequest(ctx, http.MethodGet, "/v1/users", nil)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to list users: %s", err))
		return
	}

	if statusCode != http.StatusOK {
		switch statusCode {
		case http.StatusUnauthorized:
			resp.Diagnostics.AddError("Unauthorized", "Check provider credentials (username/password)")
		case http.StatusForbidden:
			resp.Diagnostics.AddError("Forbidden", "Admin privileges required")
		default:
			resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unexpected status code %d listing users: %s", statusCode, string(respBody)))
		}
		return
	}

	var users []apiUserResponse
	if err := json.Unmarshal(respBody, &users); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Failed to parse users response: %s", err))
		return
	}

	var foundUser *apiUserResponse
	for _, u := range users {
		if u.Username == state.Username.ValueString() {
			foundUser = &u
			break
		}
	}

	if foundUser == nil {
		tflog.Debug(ctx, "User not found, removing from state", map[string]interface{}{"username": state.Username.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = state.Username
	state.Role = types.StringValue(foundUser.Role)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := apiUserAddOrUpdateRequest{
		Username: plan.Username.ValueString(),
		Tier:     plan.Tier.ValueString(),
	}

	respBody, statusCode, err := r.client.AdminRequest(ctx, http.MethodPut, "/v1/users", body)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to update user: %s", err))
		return
	}

	if statusCode != http.StatusOK {
		switch statusCode {
		case http.StatusBadRequest:
			resp.Diagnostics.AddError("Bad Request", fmt.Sprintf("ntfy returned 400: %s", string(respBody)))
		case http.StatusUnauthorized:
			resp.Diagnostics.AddError("Unauthorized", "Check provider credentials (username/password)")
		case http.StatusForbidden:
			resp.Diagnostics.AddError("Forbidden", "Admin privileges required")
		case http.StatusNotFound:
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("User %s not found", plan.Username.ValueString()))
		case http.StatusInternalServerError:
			resp.Diagnostics.AddError("Server Error", fmt.Sprintf("ntfy server error: %s", string(respBody)))
		default:
			resp.Diagnostics.AddError("Unexpected Error", fmt.Sprintf("HTTP %d: %s", statusCode, string(respBody)))
		}
		return
	}

	tflog.Debug(ctx, "Updated ntfy user", map[string]interface{}{"username": plan.Username.ValueString(), "tier": plan.Tier.ValueString()})

	// Preserve password and role from state (not returned by PUT)
	plan.Password = state.Password
	plan.Role = state.Role
	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := apiUserDeleteRequest{
		Username: state.Username.ValueString(),
	}

	respBody, statusCode, err := r.client.AdminRequest(ctx, http.MethodDelete, "/v1/users", body)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to delete user: %s", err))
		return
	}

	if statusCode != http.StatusOK {
		switch statusCode {
		case http.StatusUnauthorized:
			resp.Diagnostics.AddError("Unauthorized", "Check provider credentials (username/password)")
		case http.StatusForbidden:
			resp.Diagnostics.AddError("Forbidden", "Admin privileges required")
		case http.StatusNotFound:
			// User already gone, treat as success
			tflog.Debug(ctx, "User already deleted", map[string]interface{}{"username": state.Username.ValueString()})
			return
		default:
			resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unexpected status code %d deleting user: %s", statusCode, string(respBody)))
		}
		return
	}

	tflog.Debug(ctx, "Deleted ntfy user", map[string]interface{}{"username": state.Username.ValueString()})
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by username.
	// Password cannot be read from API — it will be unknown after import.
	// The subsequent Read() will populate role from the API.
	username := req.ID

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), username)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), username)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("password"), "")...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tier"), "")...)
}

func generatePassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}
