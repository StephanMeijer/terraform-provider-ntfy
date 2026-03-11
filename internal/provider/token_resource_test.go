package provider

import (
	"context"
	"regexp"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestTokenResource_Schema verifies all expected attributes exist in the schema.
func TestTokenResource_Schema(t *testing.T) {
	r := &TokenResource{}
	schemaReq := fwresource.SchemaRequest{}
	schemaResp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema has errors: %v", schemaResp.Diagnostics)
	}
	for _, attr := range []string{"id", "username", "password", "label", "expires", "token"} {
		if _, ok := schemaResp.Schema.Attributes[attr]; !ok {
			t.Errorf("Schema missing %q attribute", attr)
		}
	}
}

// TestTokenResource_Create_Success verifies a token is created and id=token in state.
func TestTokenResource_Create_Success(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health": {statusCode: 200, body: map[string]bool{"healthy": true}},
		"POST /v1/account/token": {statusCode: 200, body: apiAccountTokenResponse{
			Token: "tk_testtoken123",
			Label: "test",
		}},
		"GET /v1/account":          {statusCode: 200, body: map[string]string{"username": testUsername}},
		"DELETE /v1/account/token": {statusCode: 200},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "ntfy_token" "test" {
					username = "testuser"
					password = "testpass"
					label    = "test"
					expires  = 0
				}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("ntfy_token.test", "token", "tk_testtoken123"),
					resource.TestCheckResourceAttr("ntfy_token.test", "id", "tk_testtoken123"),
					resource.TestCheckResourceAttr("ntfy_token.test", "label", "test"),
				),
			},
		},
	})
}

// TestTokenResource_Create_Unauthorized verifies a 401 on POST produces an error diagnostic.
func TestTokenResource_Create_Unauthorized(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":         {statusCode: 200, body: map[string]bool{"healthy": true}},
		"POST /v1/account/token": {statusCode: 401, body: map[string]string{"error": "unauthorized"}},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "ntfy_token" "test" {
					username = "testuser"
					password = "wrongpass"
					label    = "test"
					expires  = 0
				}`,
				ExpectError: regexp.MustCompile(`Unauthorized`),
			},
		},
	})
}

// TestTokenResource_Read_Valid verifies state is preserved when GET /v1/account returns 200.
func TestTokenResource_Read_Valid(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health": {statusCode: 200, body: map[string]bool{"healthy": true}},
		"POST /v1/account/token": {statusCode: 200, body: apiAccountTokenResponse{
			Token: "tk_readtest456",
			Label: "readtest",
		}},
		"GET /v1/account":          {statusCode: 200, body: map[string]string{"username": testUsername}},
		"DELETE /v1/account/token": {statusCode: 200},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "ntfy_token" "test" {
					username = "testuser"
					password = "testpass"
					label    = "readtest"
					expires  = 0
				}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("ntfy_token.test", "token", "tk_readtest456"),
					resource.TestCheckResourceAttr("ntfy_token.test", "id", "tk_readtest456"),
				),
			},
		},
	})
}

// TestTokenResource_Read_Expired verifies that a 401 on GET /v1/account removes the resource from state.
func TestTokenResource_Read_Expired(t *testing.T) {
	mockSrv := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":           {statusCode: 200, body: map[string]bool{"healthy": true}},
		"POST /v1/account/token":   {statusCode: 200, body: apiAccountTokenResponse{Token: "tk_expiredtoken", Label: "expired"}},
		"GET /v1/account":          {statusCode: 401},
		"DELETE /v1/account/token": {statusCode: 200},
	})

	t.Setenv("NTFY_URL", mockSrv.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "ntfy_token" "test" {
					username = "testuser"
					password = "testpass"
					label    = "expired"
					expires  = 0
				}`,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestTokenResource_Delete_Success verifies clean deletion with 200 response.
func TestTokenResource_Delete_Success(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health": {statusCode: 200, body: map[string]bool{"healthy": true}},
		"POST /v1/account/token": {statusCode: 200, body: apiAccountTokenResponse{
			Token: "tk_deletetest789",
			Label: "deletetest",
		}},
		"GET /v1/account":          {statusCode: 200, body: map[string]string{"username": testUsername}},
		"DELETE /v1/account/token": {statusCode: 200},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "ntfy_token" "test" {
					username = "testuser"
					password = "testpass"
					label    = "deletetest"
					expires  = 0
				}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("ntfy_token.test", "token", "tk_deletetest789"),
				),
			},
		},
	})
}

func TestTokenResource_Configure_Error(t *testing.T) {
	r := &TokenResource{}
	req := fwresource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &fwresource.ConfigureResponse{}
	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Expected error diagnostic for invalid provider data, got none")
	}
}
