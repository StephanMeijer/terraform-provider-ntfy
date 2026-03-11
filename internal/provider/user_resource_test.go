package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestUserResource_Schema(t *testing.T) {
	r := &UserResource{}
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)

	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema has errors: %v", schemaResp.Diagnostics)
	}

	if _, ok := schemaResp.Schema.Attributes["id"]; !ok {
		t.Error("Schema missing 'id' attribute")
	}
	if _, ok := schemaResp.Schema.Attributes["username"]; !ok {
		t.Error("Schema missing 'username' attribute")
	}
	if _, ok := schemaResp.Schema.Attributes["password"]; !ok {
		t.Error("Schema missing 'password' attribute")
	}
	if _, ok := schemaResp.Schema.Attributes["role"]; !ok {
		t.Error("Schema missing 'role' attribute")
	}
	if _, ok := schemaResp.Schema.Attributes["tier"]; !ok {
		t.Error("Schema missing 'tier' attribute")
	}
}

func TestUserResource_Create_Success(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":   {statusCode: 200, body: map[string]bool{"healthy": true}},
		"POST /v1/users":   {statusCode: 200},
		"GET /v1/users":    {statusCode: 200, body: []apiUserResponse{{Username: "testcreate", Role: "user"}}},
		"DELETE /v1/users": {statusCode: 200},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: `resource "ntfy_user" "test" { username = "testcreate" }`,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("ntfy_user.test", "username", "testcreate"),
					tfresource.TestCheckResourceAttr("ntfy_user.test", "id", "testcreate"),
					tfresource.TestCheckResourceAttr("ntfy_user.test", "role", "user"),
					tfresource.TestCheckResourceAttrSet("ntfy_user.test", "password"),
				),
			},
		},
	})
}

func TestUserResource_Create_Conflict(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health": {statusCode: 200, body: map[string]bool{"healthy": true}},
		"POST /v1/users": {statusCode: 409, body: map[string]string{"error": "conflict"}},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config:      `resource "ntfy_user" "test" { username = "existing" }`,
				ExpectError: regexp.MustCompile(`Conflict|already exists`),
			},
		},
	})
}

func TestUserResource_Read_Found(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":   {statusCode: 200, body: map[string]bool{"healthy": true}},
		"POST /v1/users":   {statusCode: 200},
		"GET /v1/users":    {statusCode: 200, body: []apiUserResponse{{Username: "readuser", Role: "user"}}},
		"DELETE /v1/users": {statusCode: 200},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: `resource "ntfy_user" "test" { username = "readuser" }`,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("ntfy_user.test", "username", "readuser"),
					tfresource.TestCheckResourceAttr("ntfy_user.test", "id", "readuser"),
					tfresource.TestCheckResourceAttr("ntfy_user.test", "role", "user"),
				),
			},
		},
	})
}

func TestUserResource_Read_NotFound(t *testing.T) {
	// Test Read() directly: when user is not in the list, resource should be removed from state.
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/users": {statusCode: 200, body: []apiUserResponse{}},
	})

	client := newTestNtfyClient(server)
	r := &UserResource{client: client}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	stateVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":       tftypes.String,
			"username": tftypes.String,
			"password": tftypes.String,
			"role":     tftypes.String,
			"tier":     tftypes.String,
		},
	}, map[string]tftypes.Value{
		"id":       tftypes.NewValue(tftypes.String, "vanished"),
		"username": tftypes.NewValue(tftypes.String, "vanished"),
		"password": tftypes.NewValue(tftypes.String, "secret"),
		"role":     tftypes.NewValue(tftypes.String, "user"),
		"tier":     tftypes.NewValue(tftypes.String, ""),
	})

	readReq := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateVal,
		},
	}
	readResp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateVal,
		},
	}

	r.Read(context.Background(), readReq, readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read returned unexpected error: %v", readResp.Diagnostics)
	}

	// After RemoveResource, the state raw value should be null
	if !readResp.State.Raw.IsNull() {
		t.Error("Expected state to be removed (null) when user not found, but state is still set")
	}
}

func TestUserResource_Delete_Success(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":   {statusCode: 200, body: map[string]bool{"healthy": true}},
		"POST /v1/users":   {statusCode: 200},
		"GET /v1/users":    {statusCode: 200, body: []apiUserResponse{{Username: "deleteuser", Role: "user"}}},
		"DELETE /v1/users": {statusCode: 200},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: `resource "ntfy_user" "test" { username = "deleteuser" }`,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("ntfy_user.test", "username", "deleteuser"),
				),
			},
		},
	})
}

func TestUserResource_Delete_Unauthorized(t *testing.T) {
	// Test Delete() directly: when DELETE returns 401, should get error diagnostic.
	server := newMockNtfyServer(t, map[string]mockResponse{
		"DELETE /v1/users": {statusCode: 401, body: map[string]string{"error": "unauthorized"}},
	})

	client := newTestNtfyClient(server)
	r := &UserResource{client: client}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	stateVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":       tftypes.String,
			"username": tftypes.String,
			"password": tftypes.String,
			"role":     tftypes.String,
			"tier":     tftypes.String,
		},
	}, map[string]tftypes.Value{
		"id":       tftypes.NewValue(tftypes.String, "nodeluser"),
		"username": tftypes.NewValue(tftypes.String, "nodeluser"),
		"password": tftypes.NewValue(tftypes.String, "secret"),
		"role":     tftypes.NewValue(tftypes.String, "user"),
		"tier":     tftypes.NewValue(tftypes.String, ""),
	})

	deleteReq := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateVal,
		},
	}
	deleteResp := &resource.DeleteResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateVal,
		},
	}

	r.Delete(context.Background(), deleteReq, deleteResp)

	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("Expected error diagnostic for unauthorized delete, got none")
	}

	found := false
	for _, d := range deleteResp.Diagnostics.Errors() {
		if d.Summary() == "Unauthorized" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'Unauthorized' error, got: %v", deleteResp.Diagnostics)
	}
}

func TestUserResource_Update_Tier(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":   {statusCode: 200, body: map[string]bool{"healthy": true}},
		"POST /v1/users":   {statusCode: 200},
		"GET /v1/users":    {statusCode: 200, body: []apiUserResponse{{Username: "testuser", Role: "user"}}},
		"PUT /v1/users":    {statusCode: 200},
		"DELETE /v1/users": {statusCode: 200},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: `resource "ntfy_user" "test" { username = "testuser" }`,
				Check:  tfresource.TestCheckResourceAttr("ntfy_user.test", "username", "testuser"),
			},
			{
				Config: `
resource "ntfy_user" "test" {
  username = "testuser"
  tier     = "pro"
}`,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("ntfy_user.test", "username", "testuser"),
					tfresource.TestCheckResourceAttr("ntfy_user.test", "tier", "pro"),
				),
			},
		},
	})
}

func TestUserResource_Update_TierRemoved(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":   {statusCode: 200, body: map[string]bool{"healthy": true}},
		"POST /v1/users":   {statusCode: 200},
		"GET /v1/users":    {statusCode: 200, body: []apiUserResponse{{Username: "testuser", Role: "user"}}},
		"PUT /v1/users":    {statusCode: 200},
		"DELETE /v1/users": {statusCode: 200},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: `
resource "ntfy_user" "test" {
  username = "testuser"
  tier     = "pro"
}`,
				Check: tfresource.TestCheckResourceAttr("ntfy_user.test", "tier", "pro"),
			},
			{
				Config: `resource "ntfy_user" "test" { username = "testuser" }`,
				Check:  tfresource.TestCheckResourceAttr("ntfy_user.test", "username", "testuser"),
			},
		},
	})
}

func TestUserResource_Configure_Error(t *testing.T) {
	r := &UserResource{}
	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Expected error diagnostic for invalid provider data, got none")
	}
}
