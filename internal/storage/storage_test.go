package storage

import (
	"errors"
	"testing"
)

func TestFSStoreRoundTripAndETag(t *testing.T) {
	store, err := NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateCollection("alice", Calendars, "personal"); err != nil {
		t.Fatal(err)
	}
	data := []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n")
	first, existed, err := store.PutResource("alice", Calendars, "personal", "event.ics", data)
	if err != nil {
		t.Fatal(err)
	}
	if existed {
		t.Fatal("new resource unexpectedly existed")
	}
	got, err := store.GetResource("alice", Calendars, "personal", "event.ics")
	if err != nil {
		t.Fatal(err)
	}
	if got.ETag != first.ETag || string(got.Data) != string(data) {
		t.Fatalf("round trip mismatch: %#v", got)
	}
	_, existed, err = store.PutResource("alice", Calendars, "personal", "event.ics", data)
	if err != nil {
		t.Fatal(err)
	}
	if !existed {
		t.Fatal("updated resource should report existing")
	}
}

func TestFSStoreRejectsUnsafeSegments(t *testing.T) {
	store, err := NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"..", "../x", ".hidden", "a/b"} {
		if err := store.CreateCollection("alice", Calendars, name); !errors.Is(err, ErrInvalidSegment) {
			t.Fatalf("expected invalid segment for %q, got %v", name, err)
		}
	}
	if _, _, err := store.PutResource("alice", Calendars, "personal", "../evil.ics", []byte("x")); !errors.Is(err, ErrInvalidSegment) {
		t.Fatalf("expected invalid resource, got %v", err)
	}
}
