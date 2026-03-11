package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserDataSource_basic(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig("acctest_ds_user"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.ntfy_user.test", "username", "acctest_ds_user"),
					resource.TestCheckResourceAttrSet("data.ntfy_user.test", "role"),
					resource.TestCheckResourceAttr("data.ntfy_user.test", "id", "acctest_ds_user"),
				),
			},
		},
	})
}

func testAccUserDataSourceConfig(username string) string {
	return fmt.Sprintf(`
resource "ntfy_user" "setup" {
  username = %q
}

data "ntfy_user" "test" {
  username   = ntfy_user.setup.username
  depends_on = [ntfy_user.setup]
}
`, username)
}
