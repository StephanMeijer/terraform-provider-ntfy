package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"ntfy": providerserver.NewProtocol6WithError(New()),
}

func TestProviderSchema(t *testing.T) {
	p := New()
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	attrs := resp.Schema.Attributes
	if _, ok := attrs["url"]; !ok {
		t.Error("expected 'url' attribute to exist")
	}
	if _, ok := attrs["username"]; !ok {
		t.Error("expected 'username' attribute to exist")
	}
	if _, ok := attrs["password"]; !ok {
		t.Error("expected 'password' attribute to exist")
	}
}

func TestProvider_Configure(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/health" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		p := New()
		schemaResp := &provider.SchemaResponse{}
		p.Schema(ctx, provider.SchemaRequest{}, schemaResp)

		configValue := tftypes.NewValue(
			tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"url":      tftypes.String,
					"username": tftypes.String,
					"password": tftypes.String,
				},
			},
			map[string]tftypes.Value{
				"url":      tftypes.NewValue(tftypes.String, srv.URL),
				"username": tftypes.NewValue(tftypes.String, "admin"),
				"password": tftypes.NewValue(tftypes.String, "admin"),
			},
		)

		req := provider.ConfigureRequest{
			Config: tfsdk.Config{
				Raw:    configValue,
				Schema: schemaResp.Schema,
			},
		}
		resp := &provider.ConfigureResponse{}
		p.Configure(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("configure diagnostics: %v", resp.Diagnostics)
		}
	})

	t.Run("connection error", func(t *testing.T) {
		p := New()
		schemaResp := &provider.SchemaResponse{}
		p.Schema(ctx, provider.SchemaRequest{}, schemaResp)

		configValue := tftypes.NewValue(
			tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"url":      tftypes.String,
					"username": tftypes.String,
					"password": tftypes.String,
				},
			},
			map[string]tftypes.Value{
				"url":      tftypes.NewValue(tftypes.String, "http://nonexistent:12345"),
				"username": tftypes.NewValue(tftypes.String, "admin"),
				"password": tftypes.NewValue(tftypes.String, "admin"),
			},
		)

		req := provider.ConfigureRequest{
			Config: tfsdk.Config{
				Raw:    configValue,
				Schema: schemaResp.Schema,
			},
		}
		resp := &provider.ConfigureResponse{}
		p.Configure(ctx, req, resp)

		if !resp.Diagnostics.HasError() {
			t.Fatal("expected error but got none")
		}
	})

	t.Run("auth failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer srv.Close()

		p := New()
		schemaResp := &provider.SchemaResponse{}
		p.Schema(ctx, provider.SchemaRequest{}, schemaResp)

		configValue := tftypes.NewValue(
			tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"url":      tftypes.String,
					"username": tftypes.String,
					"password": tftypes.String,
				},
			},
			map[string]tftypes.Value{
				"url":      tftypes.NewValue(tftypes.String, srv.URL),
				"username": tftypes.NewValue(tftypes.String, "admin"),
				"password": tftypes.NewValue(tftypes.String, "wrong"),
			},
		)

		req := provider.ConfigureRequest{
			Config: tfsdk.Config{
				Raw:    configValue,
				Schema: schemaResp.Schema,
			},
		}
		resp := &provider.ConfigureResponse{}
		p.Configure(ctx, req, resp)

		if !resp.Diagnostics.HasError() {
			t.Fatal("expected error but got none")
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		p := New()
		schemaResp := &provider.SchemaResponse{}
		p.Schema(ctx, provider.SchemaRequest{}, schemaResp)

		configValue := tftypes.NewValue(
			tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"url":      tftypes.String,
					"username": tftypes.String,
					"password": tftypes.String,
				},
			},
			map[string]tftypes.Value{
				"url":      tftypes.NewValue(tftypes.String, srv.URL),
				"username": tftypes.NewValue(tftypes.String, "admin"),
				"password": tftypes.NewValue(tftypes.String, "admin"),
			},
		)

		req := provider.ConfigureRequest{
			Config: tfsdk.Config{
				Raw:    configValue,
				Schema: schemaResp.Schema,
			},
		}
		resp := &provider.ConfigureResponse{}
		p.Configure(ctx, req, resp)

		if !resp.Diagnostics.HasError() {
			t.Fatal("expected error but got none")
		}
	})

	t.Run("env vars", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		os.Setenv("NTFY_URL", srv.URL)
		os.Setenv("NTFY_USERNAME", "env-admin")
		os.Setenv("NTFY_PASSWORD", "env-pass")
		defer func() {
			os.Unsetenv("NTFY_URL")
			os.Unsetenv("NTFY_USERNAME")
			os.Unsetenv("NTFY_PASSWORD")
		}()

		p := New()
		schemaResp := &provider.SchemaResponse{}
		p.Schema(ctx, provider.SchemaRequest{}, schemaResp)

		configValue := tftypes.NewValue(
			tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"url":      tftypes.String,
					"username": tftypes.String,
					"password": tftypes.String,
				},
			},
			map[string]tftypes.Value{
				"url":      tftypes.NewValue(tftypes.String, nil),
				"username": tftypes.NewValue(tftypes.String, nil),
				"password": tftypes.NewValue(tftypes.String, nil),
			},
		)

		req := provider.ConfigureRequest{
			Config: tfsdk.Config{
				Raw:    configValue,
				Schema: schemaResp.Schema,
			},
		}
		resp := &provider.ConfigureResponse{}
		p.Configure(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("configure diagnostics: %v", resp.Diagnostics)
		}
	})
}
