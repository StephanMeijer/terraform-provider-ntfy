package provider

import (
	"os"
	"testing"
)

// testAccPreCheck verifies required environment variables are set for acceptance tests
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set — skipping acceptance test")
	}
	if v := os.Getenv("NTFY_URL"); v == "" {
		t.Fatal("NTFY_URL must be set for acceptance tests")
	}
	if v := os.Getenv("NTFY_USERNAME"); v == "" {
		t.Fatal("NTFY_USERNAME must be set for acceptance tests")
	}
	if v := os.Getenv("NTFY_PASSWORD"); v == "" {
		t.Fatal("NTFY_PASSWORD must be set for acceptance tests")
	}
}

func testAccNtfyURL() string {
	return os.Getenv("NTFY_URL")
}

func testAccNtfyUsername() string {
	return os.Getenv("NTFY_USERNAME")
}

func testAccNtfyPassword() string {
	return os.Getenv("NTFY_PASSWORD")
}
