package provider

import (
	"regexp"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestUserResource_Import(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":   {statusCode: 200, body: map[string]bool{"healthy": true}},
		"POST /v1/users":   {statusCode: 200},
		"GET /v1/users":    {statusCode: 200, body: []apiUserResponse{{Username: "testuser", Role: "user"}}},
		"DELETE /v1/users": {statusCode: 200},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	r.UnitTest(t, r.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []r.TestStep{
			{
				Config: `resource "ntfy_user" "test" { username = "testuser" }`,
				Check:  r.TestCheckResourceAttr("ntfy_user.test", "username", "testuser"),
			},
			{
				ResourceName:      "ntfy_user.test",
				ImportState:       true,
				ImportStateVerify: false,
				ImportStateId:     "testuser",
			},
		},
	})
}

func TestAccessResource_Import(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":       {statusCode: 200, body: map[string]bool{"healthy": true}},
		"PUT /v1/users/access": {statusCode: 200},
		"GET /v1/users": {statusCode: 200, body: []apiUserResponse{
			{Username: "testuser", Role: "user", Grants: []*apiUserGrantResponse{{Topic: "alerts", Permission: "read-write"}}},
		}},
		"DELETE /v1/users/access": {statusCode: 200},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	r.UnitTest(t, r.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []r.TestStep{
			{
				Config: `
resource "ntfy_access" "test" {
  username   = "testuser"
  topic      = "alerts"
  permission = "read-write"
}`,
				Check: r.TestCheckResourceAttr("ntfy_access.test", "id", "testuser/alerts"),
			},
			{
				ResourceName:      "ntfy_access.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "testuser/alerts",
			},
		},
	})
}

func TestAccessResource_Import_InvalidID(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health":       {statusCode: 200, body: map[string]bool{"healthy": true}},
		"PUT /v1/users/access": {statusCode: 200},
		"GET /v1/users": {statusCode: 200, body: []apiUserResponse{
			{Username: "testuser", Role: "user", Grants: []*apiUserGrantResponse{{Topic: "alerts", Permission: "read-write"}}},
		}},
		"DELETE /v1/users/access": {statusCode: 200},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)

	r.UnitTest(t, r.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []r.TestStep{
			{
				Config: `
resource "ntfy_access" "test" {
  username   = "testuser"
  topic      = "alerts"
  permission = "read-write"
}`,
				Check: r.TestCheckResourceAttr("ntfy_access.test", "id", "testuser/alerts"),
			},
			{
				ResourceName:  "ntfy_access.test",
				ImportState:   true,
				ImportStateId: "invalid-no-slash",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}
