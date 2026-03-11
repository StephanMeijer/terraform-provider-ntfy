package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

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
	_ resource.Resource                = &AccessResource{}
	_ resource.ResourceWithConfigure   = &AccessResource{}
	_ resource.ResourceWithImportState = &AccessResource{}
)

type AccessResource struct {
	client *NtfyClient
}

type AccessResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Username   types.String `tfsdk:"username"`
	Topic      types.String `tfsdk:"topic"`
	Permission types.String `tfsdk:"permission"`
}

func NewAccessResource() resource.Resource {
	return &AccessResource{}
}

func (r *AccessResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access"
}

func (r *AccessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages topic access permissions for an ntfy user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier for this access grant (format: username/topic).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The username to grant access to. Use `*` or `everyone` for wildcard grants.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(\*|everyone|[a-zA-Z0-9_.-]+)$`),
						"Username must be a valid ntfy username, or * or everyone for wildcard grants",
					),
				},
			},
			"topic": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The topic to grant access to. May end with `*` for wildcard matching.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[-_A-Za-z0-9]{1,64}(\*)?$`),
						"Topic must be 1-64 chars of letters/digits/hyphens/underscores, optionally ending with *",
					),
				},
			},
			"permission": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The permission level: `read-write`, `read-only`, `write-only`, or `deny`.",
				Validators: []validator.String{
					stringvalidator.OneOf("read-write", "read-only", "write-only", "deny"),
				},
			},
		},
	}
}

func (r *AccessResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := apiAccessAllowRequest{
		Username:   plan.Username.ValueString(),
		Topic:      plan.Topic.ValueString(),
		Permission: plan.Permission.ValueString(),
	}

	respBody, statusCode, err := r.client.AdminRequest(ctx, http.MethodPut, "/v1/users/access", body)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to set access: %s", err))
		return
	}

	if statusCode != http.StatusOK {
		switch statusCode {
		case http.StatusBadRequest:
			resp.Diagnostics.AddError("Bad Request", fmt.Sprintf("ntfy returned 400: %s", string(respBody)))
		case http.StatusUnauthorized:
			resp.Diagnostics.AddError("Unauthorized", "Check provider credentials")
		case http.StatusForbidden:
			resp.Diagnostics.AddError("Forbidden", "Admin privileges required")
		case http.StatusInternalServerError:
			resp.Diagnostics.AddError("Server Error", fmt.Sprintf("ntfy server error: %s", string(respBody)))
		default:
			resp.Diagnostics.AddError("Unexpected Error", fmt.Sprintf("HTTP %d: %s", statusCode, string(respBody)))
		}
		return
	}

	plan.ID = types.StringValue(plan.Username.ValueString() + "/" + plan.Topic.ValueString())

	tflog.Debug(ctx, "Created ntfy access", map[string]interface{}{
		"username":   plan.Username.ValueString(),
		"topic":      plan.Topic.ValueString(),
		"permission": plan.Permission.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccessResourceModel
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
			resp.Diagnostics.AddError("Unauthorized", "Check provider credentials")
		case http.StatusForbidden:
			resp.Diagnostics.AddError("Forbidden", "Admin privileges required")
		case http.StatusNotFound:
			resp.State.RemoveResource(ctx)
			return
		case http.StatusInternalServerError:
			resp.Diagnostics.AddError("Server Error", fmt.Sprintf("ntfy server error: %s", string(respBody)))
		default:
			resp.Diagnostics.AddError("Unexpected Error", fmt.Sprintf("HTTP %d: %s", statusCode, string(respBody)))
		}
		return
	}

	var users []apiUserResponse
	if err := json.Unmarshal(respBody, &users); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Failed to parse users response: %s", err))
		return
	}

	found := false
	for _, u := range users {
		if u.Username == state.Username.ValueString() {
			for _, g := range u.Grants {
				if g.Topic == state.Topic.ValueString() {
					found = true
					state.Permission = types.StringValue(g.Permission)
					break
				}
			}
			break
		}
	}

	if !found {
		tflog.Debug(ctx, "Access grant not found, removing from state", map[string]interface{}{
			"username": state.Username.ValueString(),
			"topic":    state.Topic.ValueString(),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(state.Username.ValueString() + "/" + state.Topic.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := apiAccessAllowRequest{
		Username:   plan.Username.ValueString(),
		Topic:      plan.Topic.ValueString(),
		Permission: plan.Permission.ValueString(),
	}

	respBody, statusCode, err := r.client.AdminRequest(ctx, http.MethodPut, "/v1/users/access", body)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to update access: %s", err))
		return
	}

	if statusCode != http.StatusOK {
		switch statusCode {
		case http.StatusBadRequest:
			resp.Diagnostics.AddError("Bad Request", fmt.Sprintf("ntfy returned 400: %s", string(respBody)))
		case http.StatusUnauthorized:
			resp.Diagnostics.AddError("Unauthorized", "Check provider credentials")
		case http.StatusForbidden:
			resp.Diagnostics.AddError("Forbidden", "Admin privileges required")
		case http.StatusInternalServerError:
			resp.Diagnostics.AddError("Server Error", fmt.Sprintf("ntfy server error: %s", string(respBody)))
		default:
			resp.Diagnostics.AddError("Unexpected Error", fmt.Sprintf("HTTP %d: %s", statusCode, string(respBody)))
		}
		return
	}

	plan.ID = types.StringValue(plan.Username.ValueString() + "/" + plan.Topic.ValueString())

	tflog.Debug(ctx, "Updated ntfy access", map[string]interface{}{
		"username":   plan.Username.ValueString(),
		"topic":      plan.Topic.ValueString(),
		"permission": plan.Permission.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'username/topic', got: %s", req.ID),
		)
		return
	}

	username := parts[0]
	topic := parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), username+"/"+topic)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), username)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("topic"), topic)...)
}

func (r *AccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := apiAccessResetRequest{
		Username: state.Username.ValueString(),
		Topic:    state.Topic.ValueString(),
	}

	respBody, statusCode, err := r.client.AdminRequest(ctx, http.MethodDelete, "/v1/users/access", body)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to delete access: %s", err))
		return
	}

	if statusCode != http.StatusOK {
		switch statusCode {
		case http.StatusBadRequest:
			resp.Diagnostics.AddError("Bad Request", fmt.Sprintf("ntfy returned 400: %s", string(respBody)))
		case http.StatusUnauthorized:
			resp.Diagnostics.AddError("Unauthorized", "Check provider credentials")
		case http.StatusForbidden:
			resp.Diagnostics.AddError("Forbidden", "Admin privileges required")
		case http.StatusNotFound:
			// Already deleted, nothing to do
			return
		case http.StatusInternalServerError:
			resp.Diagnostics.AddError("Server Error", fmt.Sprintf("ntfy server error: %s", string(respBody)))
		default:
			resp.Diagnostics.AddError("Unexpected Error", fmt.Sprintf("HTTP %d: %s", statusCode, string(respBody)))
		}
		return
	}

	tflog.Debug(ctx, "Deleted ntfy access", map[string]interface{}{
		"username": state.Username.ValueString(),
		"topic":    state.Topic.ValueString(),
	})
}
