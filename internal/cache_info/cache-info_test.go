package cache_info

import (
	"testing"
)

func TestUnmarshal(t *testing.T) {
	t.Run("parses valid input", func(t *testing.T) {
		input := []byte("StoreDir: /nix/store\nWantMassQuery: 1\nPriority: 50\n")
		info := NewCacheInfo()

		if err := Unmarshal(input, info); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if info.StoreDir != "/nix/store" {
			t.Errorf("StoreDir: got %q, want %q", info.StoreDir, "/nix/store")
		}
		if info.WantMassQuery != 1 {
			t.Errorf("WantMassQuery: got %d, want %d", info.WantMassQuery, 1)
		}
		if info.Priority != 50 {
			t.Errorf("Priority: got %d, want %d", info.Priority, 50)
		}
	})

	t.Run("parses partial input, missing fields left as default values", func(t *testing.T) {
		input := []byte("StoreDir: /nix/store\n")
		info := NewCacheInfo()

		if err := Unmarshal(input, info); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if info.StoreDir != "/nix/store" {
			t.Errorf("StoreDir: got %q, want %q", info.StoreDir, "/nix/store")
		}
		if info.WantMassQuery != 1 {
			t.Errorf("WantMassQuery: got %d, want 0", info.WantMassQuery)
		}
		if info.Priority != DefaultPriority {
			t.Errorf("Priority: got %d, want 0", info.Priority)
		}
	})

	t.Run("ignores unknown keys", func(t *testing.T) {
		input := []byte("StoreDir: /nix/store\nUnknownKey: somevalue\nPriority: 10\n")
		info := NewCacheInfo()

		if err := Unmarshal(input, info); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if info.StoreDir != "/nix/store" {
			t.Errorf("StoreDir: got %q, want %q", info.StoreDir, "/nix/store")
		}
		if info.Priority != 10 {
			t.Errorf("Priority: got %d, want 10", info.Priority)
		}
	})

	t.Run("ignores lines without separator", func(t *testing.T) {
		input := []byte("this line has no colon-space separator\nStoreDir: /nix/store\n")
		info := NewCacheInfo()

		if err := Unmarshal(input, info); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if info.StoreDir != "/nix/store" {
			t.Errorf("StoreDir: got %q, want %q", info.StoreDir, "/nix/store")
		}
	})

	t.Run("returns error on invalid WantMassQuery value", func(t *testing.T) {
		input := []byte("WantMassQuery: notanumber\n")
		info := NewCacheInfo()

		if err := Unmarshal(input, info); err == nil {
			t.Fatal("expected error for invalid WantMassQuery, got nil")
		}
	})

	t.Run("returns error on invalid Priority value", func(t *testing.T) {
		input := []byte("Priority: notanumber\n")
		info := NewCacheInfo()

		if err := Unmarshal(input, info); err == nil {
			t.Fatal("expected error for invalid Priority, got nil")
		}
	})
}

func TestCacheInfoString(t *testing.T) {
	ci := &CacheInfo{
		StoreDir:      "/nix/store",
		WantMassQuery: 1,
		Priority:      50,
	}

	got := ci.String()
	want := "StoreDir: /nix/store\nWantMassQuery: 1\nPriority: 50\n"

	if got != want {
		t.Errorf("String():\ngot:  %q\nwant: %q", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	original := &CacheInfo{
		StoreDir:      "/nix/store",
		WantMassQuery: 1,
		Priority:      50,
	}

	serialized := original.String()

	var parsed CacheInfo
	if err := Unmarshal([]byte(serialized), &parsed); err != nil {
		t.Fatalf("unexpected error during unmarshal: %v", err)
	}

	if parsed.StoreDir != original.StoreDir {
		t.Errorf("StoreDir: got %q, want %q", parsed.StoreDir, original.StoreDir)
	}
	if parsed.WantMassQuery != original.WantMassQuery {
		t.Errorf("WantMassQuery: got %d, want %d", parsed.WantMassQuery, original.WantMassQuery)
	}
	if parsed.Priority != original.Priority {
		t.Errorf("Priority: got %d, want %d", parsed.Priority, original.Priority)
	}
}
