# Changelog

## 1.0.0 (2026-03-11)


### Features

* initial release ([c531eb5](https://github.com/StephanMeijer/terraform-provider-ntfy/commit/c531eb5a4047405882662452ab7cc82bd54bfae8))

## v0.1.0 (Unreleased)

### Features

- **Hardened Resources**: 3 resources (`ntfy_user`, `ntfy_access`, `ntfy_token`) with `id` attributes, schema validators, and comprehensive error handling.
- **Data Sources**: 2 new data sources (`ntfy_user`, `ntfy_access`) for reading existing server state.
- **Import Support**: Support for importing `ntfy_user` (by username) and `ntfy_access` (by `username/topic`).
- **User Tiers**: Support for the `tier` attribute on `ntfy_user` via the `PUT /v1/users` API.
- **Reliability**: HTTP client with 30s timeout and context propagation for all requests.
- **Validation**: Provider configuration validates server connectivity on initialization.
- **Testing**: Comprehensive unit test suite with mocked HTTP and acceptance test suite with Docker-based ntfy server.
- **Documentation**: Auto-generated documentation using `terraform-plugin-docs`.
- **Releases**: GPG-signed releases for the Terraform Registry.
