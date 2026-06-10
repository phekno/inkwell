package cognito

import (
	"os"
	"testing"
	"time"
)

func TestSaveLoadClearSession(t *testing.T) {
	dir := t.TempDir()
	// UserConfigDir uses different env vars per OS; override HOME and
	// XDG_CONFIG_HOME so both Linux and macOS land in the temp dir.
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)

	if got, err := LoadSession(); err != nil || got != nil {
		t.Fatalf("expected (nil, nil), got (%v, %v)", got, err)
	}

	in := &Session{
		IDToken:      "id-token",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Email:        "josh@example.com",
	}
	if err := SaveSession(in); err != nil {
		t.Fatal(err)
	}

	p, err := sessionPath()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("perm = %o want 0o600", mode)
	}

	out, err := LoadSession()
	if err != nil {
		t.Fatal(err)
	}
	if out.IDToken != in.IDToken || out.Email != in.Email {
		t.Fatalf("round-trip mismatch: %+v vs %+v", out, in)
	}

	if err := ClearSession(); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadSession(); err != nil || got != nil {
		t.Fatalf("after Clear expected (nil, nil), got (%v, %v)", got, err)
	}
	// Clearing a missing session is a no-op.
	if err := ClearSession(); err != nil {
		t.Fatalf("idempotent clear: %v", err)
	}
}
