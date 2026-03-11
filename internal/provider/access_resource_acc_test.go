package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAccessResource_basic(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccessResourceConfig("acctest_access_user", "alerts", "read-write"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("ntfy_access.test", "username", "acctest_access_user"),
					resource.TestCheckResourceAttr("ntfy_access.test", "topic", "alerts"),
					resource.TestCheckResourceAttr("ntfy_access.test", "permission", "read-write"),
					resource.TestCheckResourceAttr("ntfy_access.test", "id", "acctest_access_user/alerts"),
				),
			},
			{
				// Update permission
				Config: testAccAccessResourceConfig("acctest_access_user", "alerts", "read-only"),
				Check:  resource.TestCheckResourceAttr("ntfy_access.test", "permission", "read-only"),
			},
		},
	})
}

func TestAccAccessResource_import(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccessResourceConfig("acctest_import_access", "news", "write-only"),
				Check:  resource.TestCheckResourceAttr("ntfy_access.test", "id", "acctest_import_access/news"),
			},
			{
				ResourceName:      "ntfy_access.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "acctest_import_access/news",
			},
		},
	})
}

func testAccAccessResourceConfig(username, topic, permission string) string {
	return fmt.Sprintf(`
provider "ntfy" {}

resource "ntfy_user" "setup" {
  username = %q
}

resource "ntfy_access" "test" {
  username   = ntfy_user.setup.username
  topic      = %q
  permission = %q
  depends_on = [ntfy_user.setup]
}
`, username, topic, permission)
}
