package config

import "testing"

func TestLoopbackListen(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:5232", "[::1]:5232", "localhost:5232"} {
		if !isLoopbackListen(addr) {
			t.Fatalf("expected %s to be loopback", addr)
		}
	}
	for _, addr := range []string{":5232", "0.0.0.0:5232", "192.0.2.10:5232"} {
		if isLoopbackListen(addr) {
			t.Fatalf("expected %s to be non-loopback", addr)
		}
	}
}

func TestDevelopmentPrincipalIDValidation(t *testing.T) {
	for _, value := range []string{"alice", "Alice_01", "family-admin", "123"} {
		if !validDevelopmentPrincipalID(value) {
			t.Fatalf("expected %q to be a valid development principal ID", value)
		}
	}
	for _, value := range []string{"", ".", "..", ".hidden", "alice@example.com", "alice smith", "alice/slash"} {
		if validDevelopmentPrincipalID(value) {
			t.Fatalf("expected %q to be rejected as a development principal ID", value)
		}
	}
}

func TestLoadRejectsNonCanonicalDevelopmentUsername(t *testing.T) {
	t.Setenv("GOREECLOUD_DAV_LISTEN", "127.0.0.1:5232")
	t.Setenv("GOREECLOUD_DAV_DATA_DIR", t.TempDir())
	t.Setenv("GOREECLOUD_DAV_USERNAME", "alice@example.com")
	t.Setenv("GOREECLOUD_DAV_PASSWORD", "development-only")
	t.Setenv("GOREECLOUD_DAV_MAX_BODY_BYTES", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected non-canonical development username to be rejected")
	}
}
