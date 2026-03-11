package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestUserDataSource_Configure_Error(t *testing.T) {
	ds := &UserDataSource{}
	req := datasource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Expected error diagnostic for invalid provider data, got none")
	}
}

func TestUserDataSource_Read_Found(t *testing.T) {
	server := newMockNtfyServer(t, map[string]mockResponse{
		"GET /v1/health": {statusCode: 200, body: map[string]bool{"healthy": true}},
		"GET /v1/users":  {statusCode: 200, body: []apiUserResponse{{Username: "testuser", Role: "user"}}},
	})
	t.Setenv("NTFY_URL", server.URL)
	t.Setenv("NTFY_USERNAME", testUsername)
	t.Setenv("NTFY_PASSWORD", testPassword)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "ntfy_user" "test" { username = "testuser" }`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.ntfy_user.test", "username", "testuser"),
					resource.TestCheckResourceAttr("data.ntfy_user.test", "role", "user"),
					resource.TestCheckResourceAttr("data.ntfy_user.test", "id", "testuser"),
				),
			},
		},
	})
}

func TestUserDataSource_Read_NotFound(t *testing.T) {
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
				Config:      `data "ntfy_user" "test" { username = "nonexistent" }`,
				ExpectError: regexp.MustCompile("Not Found"),
			},
		},
	})
}
