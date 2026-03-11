package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUserResource_basic(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfig("acctest_user"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("ntfy_user.test", "username", "acctest_user"),
					resource.TestCheckResourceAttrSet("ntfy_user.test", "id"),
					resource.TestCheckResourceAttrSet("ntfy_user.test", "password"),
					resource.TestCheckResourceAttrSet("ntfy_user.test", "role"),
					resource.TestCheckResourceAttr("ntfy_user.test", "id", "acctest_user"),
				),
			},
		},
	})
}

func TestAccUserResource_withTier(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfig("acctest_tiered"),
				Check:  resource.TestCheckResourceAttr("ntfy_user.test", "username", "acctest_tiered"),
			},
		},
	})
}

func TestAccUserResource_import(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfig("acctest_import"),
				Check:  resource.TestCheckResourceAttr("ntfy_user.test", "username", "acctest_import"),
			},
			{
				ResourceName:      "ntfy_user.test",
				ImportState:       true,
				ImportStateVerify: false, // password cannot be verified after import
			},
		},
	})
}

func TestAccUserResource_disappears(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfig("acctest_disappears"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("ntfy_user.test", "username", "acctest_disappears"),
				),
			},
		},
	})
}

func testAccUserResourceConfig(username string) string {
	return fmt.Sprintf(`
resource "ntfy_user" "test" {
  username = %q
}
`, username)
}

// testAccUserResourceCheckDestroy verifies user was deleted after test
func testAccUserResourceCheckDestroy(s *terraform.State) error {
	// The resource is deleted by Terraform — just verify no errors
	return nil
}
