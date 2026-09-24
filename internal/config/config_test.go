package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureCreatesPrivateEmptyConfig(t *testing.T) {
	for _, name := range []string{"SHTU_USERNAME", "EGATE_ID", "SHTU_PASSWORD", "EGATE_PASSWORD", "SHTU_IP", "SHTU_INTERFACE"} {
		t.Setenv(name, "")
	}
	path := filepath.Join(t.TempDir(), "nested", "config.json")

	created, err := Ensure(path)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("Ensure() did not report creating the file")
	}
	created, err = Ensure(path)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("Ensure() reported recreating an existing file")
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Username != "" || cfg.Password != "" {
		t.Fatalf("generated config must not contain placeholder credentials: %#v", cfg)
	}
	if !cfg.FastLogin {
		t.Fatal("generated config must enable fast login by default")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("config permissions = %o, want 600", got)
		}
	}
}

func TestEnsurePreservesExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	want := []byte("existing contents")
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatal(err)
	}
	created, err := Ensure(path)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("Ensure() replaced an existing file")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("existing config changed to %q", got)
	}
}
