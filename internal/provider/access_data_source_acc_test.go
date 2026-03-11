package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAccessDataSource_basic(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccessDataSourceConfig("acctest_ds_access", "notifications", "read-only"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.ntfy_access.test", "username", "acctest_ds_access"),
					resource.TestCheckResourceAttr("data.ntfy_access.test", "topic", "notifications"),
					resource.TestCheckResourceAttr("data.ntfy_access.test", "permission", "read-only"),
					resource.TestCheckResourceAttr("data.ntfy_access.test", "id", "acctest_ds_access/notifications"),
				),
			},
		},
	})
}

func testAccAccessDataSourceConfig(username, topic, permission string) string {
	return fmt.Sprintf(`
provider "ntfy" {}

resource "ntfy_user" "setup" {
  username = %q
}

resource "ntfy_access" "setup" {
  username   = ntfy_user.setup.username
  topic      = %q
  permission = %q
  depends_on = [ntfy_user.setup]
}

data "ntfy_access" "test" {
  username   = ntfy_user.setup.username
  topic      = %q
  depends_on = [ntfy_access.setup]
}
`, username, topic, permission, topic)
}
