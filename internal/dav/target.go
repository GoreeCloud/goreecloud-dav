package dav

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/GoreeCloud/goreecloud-dav/internal/storage"
)

type objectType int

const (
	objectUnknown objectType = iota
	objectDAVRoot
	objectPrincipal
	objectHome
	objectCollection
	objectResource
)

type targetLevel int

const (
	levelUnknown targetLevel = iota
	levelDAVRoot
	levelPrincipal
	levelHome
	levelCollection
	levelResource
)

type davTarget struct {
	principal  string
	kind       storage.Kind
	collection string
	resource   string
	object     objectType
}

func parseTarget(rawPath string) (davTarget, error) {
	clean := strings.Trim(rawPath, "/")
	if clean == "dav" {
		return davTarget{object: objectDAVRoot}, nil
	}
	parts := strings.Split(clean, "/")
	if len(parts) < 3 || parts[0] != "dav" {
		return davTarget{}, fmt.Errorf("not a DAV path")
	}

	switch parts[1] {
	case "principals":
		if len(parts) != 3 {
			return davTarget{}, fmt.Errorf("invalid principal path")
		}
		return davTarget{principal: parts[2], object: objectPrincipal}, nil
	case "calendars", "addressbooks":
		kind := storage.Calendars
		if parts[1] == "addressbooks" {
			kind = storage.AddressBooks
		}
		target := davTarget{principal: parts[2], kind: kind, object: objectHome}
		if len(parts) >= 4 && parts[3] != "" {
			target.collection = parts[3]
			target.object = objectCollection
		}
		if len(parts) == 5 && parts[4] != "" {
			target.resource = parts[4]
			target.object = objectResource
		}
		if len(parts) > 5 {
			return davTarget{}, fmt.Errorf("invalid DAV depth")
		}
		return target, nil
	default:
		return davTarget{}, fmt.Errorf("unknown DAV namespace")
	}
}

func (t davTarget) level() targetLevel {
	switch t.object {
	case objectDAVRoot:
		return levelDAVRoot
	case objectPrincipal:
		return levelPrincipal
	case objectHome:
		return levelHome
	case objectCollection:
		return levelCollection
	case objectResource:
		return levelResource
	default:
		return levelUnknown
	}
}

func (t davTarget) href() string {
	switch t.object {
	case objectDAVRoot:
		return "/dav/"
	case objectPrincipal:
		return "/dav/principals/" + url.PathEscape(t.principal) + "/"
	case objectHome:
		return "/dav/" + string(t.kind) + "/" + url.PathEscape(t.principal) + "/"
	case objectCollection:
		return "/dav/" + string(t.kind) + "/" + url.PathEscape(t.principal) + "/" + url.PathEscape(t.collection) + "/"
	case objectResource:
		return "/dav/" + string(t.kind) + "/" + url.PathEscape(t.principal) + "/" + url.PathEscape(t.collection) + "/" + url.PathEscape(t.resource)
	default:
		return "/dav/"
	}
}

func (t davTarget) resourceType() string {
	switch t.object {
	case objectPrincipal:
		return "principal"
	case objectHome, objectDAVRoot:
		return "collection"
	case objectCollection:
		if t.kind == storage.Calendars {
			return "calendar"
		}
		return "addressbook"
	case objectResource:
		return "resource"
	default:
		return "resource"
	}
}

func displayName(t davTarget) string {
	switch t.object {
	case objectDAVRoot:
		return "GoreeCloud DAV"
	case objectPrincipal:
		return t.principal
	case objectHome:
		if t.kind == storage.Calendars {
			return "Calendars"
		}
		return "Address Books"
	case objectCollection:
		return t.collection
	case objectResource:
		return t.resource
	default:
		return ""
	}
}
