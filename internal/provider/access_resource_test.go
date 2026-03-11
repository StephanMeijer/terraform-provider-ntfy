package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccessResource_Schema(t *testing.T) {
	res := &AccessResource{}
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(context.Background(), schemaReq, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema has errors: %v", schemaResp.Diagnostics)
	}
	for _, attr := range []string{"id", "username", "topic", "permission"} {
		if _, ok := schemaResp.Schema.Attributes[attr]; !ok {
			t.Errorf("Schema missing %q attribute", attr)
		}
	}
}

func TestAccessResource_Create_Success(t *testing.T) {
	srv := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":       {statusCode: 200, body: map[string]bool{"healthy": true}},
		"PUT /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
		"GET /v1/users": {statusCode: 200, body: []apiUserResponse{
			{Username: testUsername, Role: "user", Grants: []*apiUserGrantResponse{
				{Topic: testTopic, Permission: testPermission},
			}},
		}},
		"DELETE /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
	})

	r.UnitTest(t, r.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []r.TestStep{
			{
				Config: testAccessConfig(srv.URL, testUsername, testTopic, testPermission),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("ntfy_access.test", "username", testUsername),
					r.TestCheckResourceAttr("ntfy_access.test", "topic", testTopic),
					r.TestCheckResourceAttr("ntfy_access.test", "permission", testPermission),
					r.TestCheckResourceAttr("ntfy_access.test", "id", testUsername+"/"+testTopic),
				),
			},
		},
	})
}

func TestAccessResource_Create_WildcardTopic(t *testing.T) {
	srv := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":       {statusCode: 200, body: map[string]bool{"healthy": true}},
		"PUT /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
		"GET /v1/users": {statusCode: 200, body: []apiUserResponse{
			{Username: testUsername, Role: "user", Grants: []*apiUserGrantResponse{
				{Topic: "alerts_*", Permission: "read-only"},
			}},
		}},
		"DELETE /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
	})

	r.UnitTest(t, r.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []r.TestStep{
			{
				Config: testAccessConfig(srv.URL, testUsername, "alerts_*", "read-only"),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("ntfy_access.test", "topic", "alerts_*"),
					r.TestCheckResourceAttr("ntfy_access.test", "id", testUsername+"/alerts_*"),
				),
			},
		},
	})
}

func TestAccessResource_Create_EveryoneUser(t *testing.T) {
	srv := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":       {statusCode: 200, body: map[string]bool{"healthy": true}},
		"PUT /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
		"GET /v1/users": {statusCode: 200, body: []apiUserResponse{
			{Username: "*", Role: "anonymous", Grants: []*apiUserGrantResponse{
				{Topic: testTopic, Permission: "deny"},
			}},
		}},
		"DELETE /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
	})

	r.UnitTest(t, r.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []r.TestStep{
			{
				Config: testAccessConfig(srv.URL, "*", testTopic, "deny"),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("ntfy_access.test", "username", "*"),
					r.TestCheckResourceAttr("ntfy_access.test", "id", "*/"+testTopic),
				),
			},
		},
	})
}

func TestAccessResource_Read_Found(t *testing.T) {
	srv := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":       {statusCode: 200, body: map[string]bool{"healthy": true}},
		"PUT /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
		"GET /v1/users": {statusCode: 200, body: []apiUserResponse{
			{Username: testUsername, Role: "user", Grants: []*apiUserGrantResponse{
				{Topic: testTopic, Permission: "write-only"},
			}},
		}},
		"DELETE /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
	})

	r.UnitTest(t, r.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []r.TestStep{
			{
				Config: testAccessConfig(srv.URL, testUsername, testTopic, "write-only"),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("ntfy_access.test", "permission", "write-only"),
				),
			},
		},
	})
}

func TestAccessResource_Read_NotFound(t *testing.T) {
	// Mock returns user but without the matching grant — resource should be removed
	srv := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":       {statusCode: 200, body: map[string]bool{"healthy": true}},
		"PUT /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
		// First GET returns grant (for Create), subsequent GETs return no grants (for Read)
		// Since our mock is static, we use a different approach: create with one topic,
		// but mock returns grants for a different topic so the grant isn't found on refresh
		"GET /v1/users": {statusCode: 200, body: []apiUserResponse{
			{Username: testUsername, Role: "user", Grants: []*apiUserGrantResponse{
				{Topic: "other-topic", Permission: "read-write"},
			}},
		}},
		"DELETE /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
	})

	r.UnitTest(t, r.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []r.TestStep{
			{
				Config: testAccessConfig(srv.URL, testUsername, testTopic, testPermission),
				// After Create, the Read will not find the grant (mock returns different topic),
				// so Terraform detects the resource was removed and plans to recreate.
				// ExpectNonEmptyPlan tells the framework this is expected.
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccessResource_Update_Success(t *testing.T) {
	// Step 1 mock: create with read-only
	srv1 := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":       {statusCode: 200, body: map[string]bool{"healthy": true}},
		"PUT /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
		"GET /v1/users": {statusCode: 200, body: []apiUserResponse{
			{Username: testUsername, Role: "user", Grants: []*apiUserGrantResponse{
				{Topic: testTopic, Permission: "read-only"},
			}},
		}},
		"DELETE /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
	})

	// Step 2 mock: update to read-write
	srv2 := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":       {statusCode: 200, body: map[string]bool{"healthy": true}},
		"PUT /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
		"GET /v1/users": {statusCode: 200, body: []apiUserResponse{
			{Username: testUsername, Role: "user", Grants: []*apiUserGrantResponse{
				{Topic: testTopic, Permission: "read-write"},
			}},
		}},
		"DELETE /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
	})

	r.UnitTest(t, r.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []r.TestStep{
			{
				Config: testAccessConfig(srv1.URL, testUsername, testTopic, "read-only"),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("ntfy_access.test", "permission", "read-only"),
				),
			},
			{
				Config: testAccessConfig(srv2.URL, testUsername, testTopic, "read-write"),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("ntfy_access.test", "permission", "read-write"),
				),
			},
		},
	})
}

func TestAccessResource_Delete_Success(t *testing.T) {
	srv := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":       {statusCode: 200, body: map[string]bool{"healthy": true}},
		"PUT /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
		"GET /v1/users": {statusCode: 200, body: []apiUserResponse{
			{Username: testUsername, Role: "user", Grants: []*apiUserGrantResponse{
				{Topic: testTopic, Permission: testPermission},
			}},
		}},
		"DELETE /v1/users/access": {statusCode: 200, body: map[string]string{"success": "true"}},
	})

	r.UnitTest(t, r.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []r.TestStep{
			{
				Config: testAccessConfig(srv.URL, testUsername, testTopic, testPermission),
			},
		},
	})
	// After the test case completes, Terraform automatically calls Delete to clean up.
	// If Delete fails, the test would error.
}

func TestAccessResource_Configure_Error(t *testing.T) {
	res := &AccessResource{}
	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}
	res.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Expected error diagnostic for invalid provider data, got none")
	}
}

// testAccessConfig generates a Terraform config for the ntfy_access resource
func testAccessConfig(serverURL, username, topic, permission string) string {
	return fmt.Sprintf(`
provider "ntfy" {
  url      = %q
  username = "admin"
  password = "admin"
}

resource "ntfy_access" "test" {
  username   = %q
  topic      = %q
  permission = %q
}
`, serverURL, username, topic, permission)
}
