package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTokenResource_basic(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenResourceConfig("acctest_token_user", "test-token"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("ntfy_token.test", "label", "test-token"),
					resource.TestCheckResourceAttrSet("ntfy_token.test", "token"),
					resource.TestCheckResourceAttrSet("ntfy_token.test", "id"),
					resource.TestCheckResourceAttr("ntfy_token.test", "expires", "0"),
				),
			},
		},
	})
}

func TestAccTokenResource_noLabel(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenResourceConfigNoLabel("acctest_token_nolabel"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("ntfy_token.test", "token"),
					resource.TestCheckResourceAttrSet("ntfy_token.test", "id"),
					resource.TestCheckResourceAttr("ntfy_token.test", "expires", "0"),
				),
			},
		},
	})
}

func testAccTokenResourceConfig(username, label string) string {
	return fmt.Sprintf(`
resource "ntfy_user" "setup" {
  username = %q
}

resource "ntfy_token" "test" {
  username   = ntfy_user.setup.username
  password   = ntfy_user.setup.password
  label      = %q
  expires    = 0
  depends_on = [ntfy_user.setup]
}
`, username, label)
}

func testAccTokenResourceConfigNoLabel(username string) string {
	return fmt.Sprintf(`
resource "ntfy_user" "setup" {
  username = %q
}

resource "ntfy_token" "test" {
  username   = ntfy_user.setup.username
  password   = ntfy_user.setup.password
  expires    = 0
  depends_on = [ntfy_user.setup]
}
`, username)
}
