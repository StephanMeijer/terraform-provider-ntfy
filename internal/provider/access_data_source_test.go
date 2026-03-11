package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccessDataSource_Configure_Error(t *testing.T) {
	ds := &AccessDataSource{}
	req := datasource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Expected error diagnostic for invalid provider data, got none")
	}
}

func TestAccessDataSource_Read_Found(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health": {statusCode: 200, body: map[string]bool{"healthy": true}},
		"GET /v1/users": {statusCode: 200, body: []apiUserResponse{
			{
				Username: "testuser",
				Role:     "user",
				Grants:   []*apiUserGrantResponse{{Topic: "alerts", Permission: "read-write"}},
			},
		}},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "ntfy_access" "test" {
  username = "testuser"
  topic    = "alerts"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.ntfy_access.test", "username", "testuser"),
					resource.TestCheckResourceAttr("data.ntfy_access.test", "topic", "alerts"),
					resource.TestCheckResourceAttr("data.ntfy_access.test", "permission", "read-write"),
					resource.TestCheckResourceAttr("data.ntfy_access.test", "id", "testuser/alerts"),
				),
			},
		},
	})
}

func TestAccessDataSource_Read_UserNotFound(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health": {statusCode: 200, body: map[string]bool{"healthy": true}},
		"GET /v1/users":  {statusCode: 200, body: []apiUserResponse{}},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "ntfy_access" "test" {
  username = "nobody"
  topic    = "alerts"
}
`,
				ExpectError: regexp.MustCompile("Not Found"),
			},
		},
	})
}

func TestAccessDataSource_Read_GrantNotFound(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health": {statusCode: 200, body: map[string]bool{"healthy": true}},
		"GET /v1/users": {statusCode: 200, body: []apiUserResponse{
			{Username: "testuser", Role: "user", Grants: []*apiUserGrantResponse{}},
		}},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "ntfy_access" "test" {
  username = "testuser"
  topic    = "nonexistent"
}
`,
				ExpectError: regexp.MustCompile("Not Found"),
			},
		},
	})
}
