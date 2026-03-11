resource "ntfy_access" "example" {
  username   = "myuser"
  topic      = "alerts"
  permission = "read-write"
}
