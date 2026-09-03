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
