package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

type Kind string

const (
	Calendars    Kind = "calendars"
	AddressBooks Kind = "addressbooks"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrAlreadyExists  = errors.New("already exists")
	ErrInvalidSegment = errors.New("invalid path segment")
	ErrNotEmpty       = errors.New("collection is not empty")
)

type Collection struct {
	Name string
	Kind Kind
}

type Resource struct {
	Name        string
	Data        []byte
	ETag        string
	ContentType string
}

type Store interface {
	CreateCollection(principal string, kind Kind, name string) error
	ListCollections(principal string, kind Kind) ([]Collection, error)
	DeleteCollection(principal string, kind Kind, name string) error
	PutResource(principal string, kind Kind, collection, name string, data []byte) (Resource, bool, error)
	GetResource(principal string, kind Kind, collection, name string) (Resource, error)
	ListResources(principal string, kind Kind, collection string) ([]Resource, error)
	DeleteResource(principal string, kind Kind, collection, name string) error
}

type FSStore struct {
	root string
}

func NewFS(root string) (*FSStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("storage root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve storage root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	return &FSStore{root: abs}, nil
}

func (s *FSStore) CreateCollection(principal string, kind Kind, name string) error {
	path, err := s.collectionPath(principal, kind, name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return ErrAlreadyExists
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (s *FSStore) ListCollections(principal string, kind Kind) ([]Collection, error) {
	base, err := s.homePath(principal, kind)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(base)
	if errors.Is(err, fs.ErrNotExist) {
		return []Collection{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]Collection, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && validSegment(entry.Name()) {
			out = append(out, Collection{Name: entry.Name(), Kind: kind})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *FSStore) DeleteCollection(principal string, kind Kind, name string) error {
	path, err := s.collectionPath(principal, kind, name)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return ErrNotFound
	case errors.Is(err, syscall.ENOTEMPTY), errors.Is(err, syscall.EEXIST):
		return ErrNotEmpty
	case err != nil:
		return err
	default:
		return nil
	}
}

func (s *FSStore) PutResource(principal string, kind Kind, collection, name string, data []byte) (Resource, bool, error) {
	path, err := s.resourcePath(principal, kind, collection, name)
	if err != nil {
		return Resource{}, false, err
	}
	collectionPath, err := s.collectionPath(principal, kind, collection)
	if err != nil {
		return Resource{}, false, err
	}
	info, err := os.Stat(collectionPath)
	if errors.Is(err, fs.ErrNotExist) {
		return Resource{}, false, ErrNotFound
	}
	if err != nil {
		return Resource{}, false, err
	}
	if !info.IsDir() {
		return Resource{}, false, ErrNotFound
	}
	_, statErr := os.Stat(path)
	existed := statErr == nil
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return Resource{}, false, statErr
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".goreecloud-dav-*")
	if err != nil {
		return Resource{}, false, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return Resource{}, false, err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return Resource{}, false, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return Resource{}, false, err
	}
	if err := tmp.Close(); err != nil {
		return Resource{}, false, err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return Resource{}, false, err
	}
	return resourceFrom(kind, name, data), existed, nil
}

func (s *FSStore) GetResource(principal string, kind Kind, collection, name string) (Resource, error) {
	path, err := s.resourcePath(principal, kind, collection, name)
	if err != nil {
		return Resource{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Resource{}, ErrNotFound
	}
	if err != nil {
		return Resource{}, err
	}
	return resourceFrom(kind, name, data), nil
}

func (s *FSStore) ListResources(principal string, kind Kind, collection string) ([]Resource, error) {
	path, err := s.collectionPath(principal, kind, collection)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	out := make([]Resource, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !validResourceName(kind, entry.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, resourceFrom(kind, entry.Name(), data))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *FSStore) DeleteResource(principal string, kind Kind, collection, name string) error {
	path, err := s.resourcePath(principal, kind, collection, name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *FSStore) homePath(principal string, kind Kind) (string, error) {
	if !validSegment(principal) || !validKind(kind) {
		return "", ErrInvalidSegment
	}
	return s.safeJoin(principal, string(kind))
}

func (s *FSStore) collectionPath(principal string, kind Kind, collection string) (string, error) {
	if !validSegment(collection) {
		return "", ErrInvalidSegment
	}
	home, err := s.homePath(principal, kind)
	if err != nil {
		return "", err
	}
	return s.safeJoinFrom(home, collection)
}

func (s *FSStore) resourcePath(principal string, kind Kind, collection, name string) (string, error) {
	if !validResourceName(kind, name) {
		return "", ErrInvalidSegment
	}
	coll, err := s.collectionPath(principal, kind, collection)
	if err != nil {
		return "", err
	}
	return s.safeJoinFrom(coll, name)
}

func (s *FSStore) safeJoin(parts ...string) (string, error) {
	return s.safeJoinFrom(s.root, parts...)
}

func (s *FSStore) safeJoinFrom(base string, parts ...string) (string, error) {
	path := filepath.Join(append([]string{base}, parts...)...)
	rel, err := filepath.Rel(s.root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", ErrInvalidSegment
	}
	return path, nil
}

func validKind(kind Kind) bool {
	return kind == Calendars || kind == AddressBooks
}

func validSegment(segment string) bool {
	if segment == "" || segment == "." || segment == ".." || strings.HasPrefix(segment, ".") {
		return false
	}
	for _, r := range segment {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func validResourceName(kind Kind, name string) bool {
	if !validSegment(strings.TrimSuffix(strings.TrimSuffix(name, ".ics"), ".vcf")) {
		return false
	}
	switch kind {
	case Calendars:
		return strings.HasSuffix(strings.ToLower(name), ".ics")
	case AddressBooks:
		return strings.HasSuffix(strings.ToLower(name), ".vcf")
	default:
		return false
	}
}

func resourceFrom(kind Kind, name string, data []byte) Resource {
	sum := sha256.Sum256(data)
	return Resource{
		Name:        name,
		Data:        append([]byte(nil), data...),
		ETag:        `"` + hex.EncodeToString(sum[:]) + `"`,
		ContentType: contentTypeFor(kind),
	}
}

func contentTypeFor(kind Kind) string {
	if kind == Calendars {
		return "text/calendar; charset=utf-8"
	}
	return "text/vcard; charset=utf-8"
}
