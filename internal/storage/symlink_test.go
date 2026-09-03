package storage

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestFSStoreRejectsSymlinkCollectionEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	store, err := NewFS(root)
	if err != nil {
		t.Fatal(err)
	}

	parent := filepath.Join(root, "alice", string(Calendars))
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(parent, "personal")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	data := []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n")
	if _, _, err := store.PutResource("alice", Calendars, "personal", "event.ics", data); !errors.Is(err, ErrInvalidSegment) {
		t.Fatalf("expected symlink collection rejection, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "event.ics")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("storage escape wrote outside root: %v", err)
	}
}

func TestFSStoreRejectsSymlinkResourceRead(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	store, err := NewFS(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateCollection("alice", Calendars, "personal"); err != nil {
		t.Fatal(err)
	}

	outsideResource := filepath.Join(outside, "secret.ics")
	if err := os.WriteFile(outsideResource, []byte("BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	resourcePath := filepath.Join(root, "alice", string(Calendars), "personal", "event.ics")
	if err := os.Symlink(outsideResource, resourcePath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if _, err := store.GetResource("alice", Calendars, "personal", "event.ics"); !errors.Is(err, ErrInvalidSegment) {
		t.Fatalf("expected symlink resource rejection, got %v", err)
	}
	if _, err := store.ListResources("alice", Calendars, "personal"); !errors.Is(err, ErrInvalidSegment) {
		t.Fatalf("expected symlink listing rejection, got %v", err)
	}
}

func TestFSStoreAcceptsCaseInsensitiveResourceExtension(t *testing.T) {
	store, err := NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateCollection("alice", Calendars, "personal"); err != nil {
		t.Fatal(err)
	}
	data := []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n")
	if _, _, err := store.PutResource("alice", Calendars, "personal", "EVENT.ICS", data); err != nil {
		t.Fatalf("uppercase calendar extension rejected: %v", err)
	}
}
