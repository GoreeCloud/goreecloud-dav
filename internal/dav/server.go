package dav

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/GoreeCloud/goreecloud-dav/internal/auth"
	"github.com/GoreeCloud/goreecloud-dav/internal/storage"
)

const reportBodyLimit = int64(1024 * 1024)

type Server struct {
	store        storage.Store
	auth         auth.Provider
	maxBodyBytes int64
}

func New(store storage.Store, provider auth.Provider, maxBodyBytes int64) *Server {
	return &Server{store: store, auth: provider, maxBodyBytes: maxBodyBytes}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/readyz", s.ready)
	mux.HandleFunc("/api/v1/status", s.status)
	mux.HandleFunc("/.well-known/caldav", wellKnownDAV)
	mux.HandleFunc("/.well-known/carddav", wellKnownDAV)
	mux.Handle("/dav/", s.requireAuth(http.HandlerFunc(s.handleDAV)))
	return securityHeaders(mux)
}

func wellKnownDAV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.Redirect(w, r, "/dav/", http.StatusTemporaryRedirect)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"service": "GoreeCloud DAV",
		"status":  "ok",
	})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"service": "GoreeCloud DAV",
		"status":  "ready",
	})
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"product":   "GoreeCloud DAV",
		"lifecycle": "active-development",
		"native":    true,
		"protocols": map[string]string{
			"webdav":  "foundation",
			"caldav":  "foundation",
			"carddav": "foundation",
		},
		"platform_integrations": map[string]string{
			"goreecloud_identity": "migration-required",
			"privacy_shield":      "migration-required",
			"wardveil_security":   "migration-required",
			"everkeep":            "migration-required",
			"goreecloud_manager":  "migration-required",
			"goreecloud_mesh":     "migration-required",
			"glaze_ui":            "not-applicable-justified-headless",
		},
	})
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := s.auth.Authenticate(r)
		if !ok {
			s.auth.Challenge(w)
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		ctx := withPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) handleDAV(w http.ResponseWriter, r *http.Request) {
	target, err := parseTarget(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		http.Error(w, "principal unavailable", http.StatusInternalServerError)
		return
	}
	if target.principal != "" && target.principal != principal.ID {
		http.Error(w, "principal mismatch", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodOptions:
		s.handleOptions(w, target)
	case "PROPFIND":
		s.handlePropfind(w, r, principal, target)
	case "MKCALENDAR":
		s.handleMkCalendar(w, principal, target)
	case "MKCOL":
		s.handleMkCol(w, principal, target)
	case http.MethodGet, http.MethodHead:
		s.handleGet(w, r, principal, target)
	case http.MethodPut:
		s.handlePut(w, r, principal, target)
	case http.MethodDelete:
		s.handleDelete(w, principal, target)
	case "REPORT":
		s.handleReport(w, r, principal, target)
	default:
		methodNotAllowed(w, "OPTIONS, PROPFIND, MKCALENDAR, MKCOL, GET, HEAD, PUT, DELETE, REPORT")
	}
}

func (s *Server) handleOptions(w http.ResponseWriter, target davTarget) {
	// Advertise only the WebDAV compliance level this foundation can currently
	// substantiate. CalDAV/CardDAV compliance tokens are intentionally withheld
	// until all applicable MUST-level requirements are implemented and tested.
	w.Header().Set("DAV", "1")
	w.Header().Set("Allow", allowedMethods(target))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePropfind(w http.ResponseWriter, r *http.Request, principal auth.Principal, target davTarget) {
	depth := strings.TrimSpace(r.Header.Get("Depth"))
	if depth == "" {
		depth = "0"
	}
	if depth != "0" && depth != "1" {
		http.Error(w, "only Depth: 0 and Depth: 1 are supported", http.StatusForbidden)
		return
	}

	responses, err := s.propertyResponses(principal, target, depth == "1")
	if err != nil {
		writeStorageError(w, err)
		return
	}
	writeMultiStatus(w, responses)
}

func (s *Server) handleMkCalendar(w http.ResponseWriter, principal auth.Principal, target davTarget) {
	if target.kind != storage.Calendars || target.collection == "" || target.resource != "" {
		http.Error(w, "MKCALENDAR requires a calendar collection path", http.StatusConflict)
		return
	}
	err := s.store.CreateCollection(principal.ID, storage.Calendars, target.collection)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleMkCol(w http.ResponseWriter, principal auth.Principal, target davTarget) {
	if target.kind != storage.AddressBooks || target.collection == "" || target.resource != "" {
		http.Error(w, "MKCOL requires an address-book collection path", http.StatusConflict)
		return
	}
	err := s.store.CreateCollection(principal.ID, storage.AddressBooks, target.collection)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request, principal auth.Principal, target davTarget) {
	if target.resource == "" || target.collection == "" || target.kind == "" {
		methodNotAllowed(w, "OPTIONS, PROPFIND, REPORT")
		return
	}
	resource, err := s.store.GetResource(principal.ID, target.kind, target.collection, target.resource)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	w.Header().Set("Content-Type", resource.ContentType)
	w.Header().Set("ETag", resource.ETag)
	w.Header().Set("Cache-Control", "private, no-cache")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resource.Data)
}

func (s *Server) handlePut(w http.ResponseWriter, r *http.Request, principal auth.Principal, target davTarget) {
	if target.resource == "" || target.collection == "" || target.kind == "" {
		http.Error(w, "PUT requires a DAV resource path", http.StatusConflict)
		return
	}

	current, getErr := s.store.GetResource(principal.ID, target.kind, target.collection, target.resource)
	exists := getErr == nil
	if getErr != nil && !errors.Is(getErr, storage.ErrNotFound) {
		writeStorageError(w, getErr)
		return
	}
	if !preconditionsSatisfied(r, current.ETag, exists) {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	body := http.MaxBytesReader(w, r.Body, s.maxBodyBytes)
	data, err := io.ReadAll(body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "resource exceeds configured maximum", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := validateResource(target.kind, data); err != nil {
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
		return
	}

	resource, existed, err := s.store.PutResource(principal.ID, target.kind, target.collection, target.resource, data)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	w.Header().Set("ETag", resource.ETag)
	if existed {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
}

func (s *Server) handleDelete(w http.ResponseWriter, principal auth.Principal, target davTarget) {
	var err error
	switch {
	case target.kind == "" || target.collection == "":
		http.Error(w, "cannot delete DAV homes or principals", http.StatusForbidden)
		return
	case target.resource != "":
		err = s.store.DeleteResource(principal.ID, target.kind, target.collection, target.resource)
	default:
		err = s.store.DeleteCollection(principal.ID, target.kind, target.collection)
	}
	if err != nil {
		writeStorageError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request, principal auth.Principal, target davTarget) {
	if target.collection == "" || target.resource != "" || target.kind == "" {
		http.Error(w, "REPORT requires a collection path", http.StatusConflict)
		return
	}
	body := http.MaxBytesReader(w, r.Body, reportBodyLimit)
	data, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, "invalid REPORT body", http.StatusBadRequest)
		return
	}

	reportName, hrefs, err := parseReport(data)
	if err != nil {
		http.Error(w, "unsupported or malformed REPORT", http.StatusBadRequest)
		return
	}
	if !reportAllowed(reportName, target.kind) {
		http.Error(w, "REPORT type is not valid for this collection", http.StatusForbidden)
		return
	}

	resources, err := s.store.ListResources(principal.ID, target.kind, target.collection)
	if err != nil {
		writeStorageError(w, err)
		return
	}

	filter := make(map[string]struct{}, len(hrefs))
	for _, href := range hrefs {
		name := path.Base(href)
		if name != "." && name != "/" && name != "" {
			filter[name] = struct{}{}
		}
	}

	responses := make([]propertyResponse, 0, len(resources))
	base := target.href()
	for _, resource := range resources {
		if strings.HasSuffix(reportName, "multiget") && len(filter) > 0 {
			if _, ok := filter[resource.Name]; !ok {
				continue
			}
		}
		responses = append(responses, propertyResponse{
			Href:         base + url.PathEscape(resource.Name),
			ResourceType: "resource",
			ETag:         resource.ETag,
			ContentType:  resource.ContentType,
			Data:         string(resource.Data),
			DataKind:     target.kind,
		})
	}
	writeMultiStatus(w, responses)
}

func (s *Server) propertyResponses(principal auth.Principal, target davTarget, includeChildren bool) ([]propertyResponse, error) {
	base := propertyResponseFor(target, principal.ID)

	switch target.level() {
	case levelResource:
		resource, err := s.store.GetResource(principal.ID, target.kind, target.collection, target.resource)
		if err != nil {
			return nil, err
		}
		base.ETag = resource.ETag
		base.ContentType = resource.ContentType
	case levelCollection:
		// Touch the collection through the storage interface so a Depth: 0
		// PROPFIND cannot report a collection that does not exist.
		if _, err := s.store.ListResources(principal.ID, target.kind, target.collection); err != nil {
			return nil, err
		}
	}

	responses := []propertyResponse{base}
	if !includeChildren {
		return responses, nil
	}

	switch target.level() {
	case levelDAVRoot:
		responses = append(responses,
			propertyResponseFor(davTarget{principal: principal.ID, object: objectPrincipal}, principal.ID),
			propertyResponseFor(davTarget{principal: principal.ID, kind: storage.Calendars, object: objectHome}, principal.ID),
			propertyResponseFor(davTarget{principal: principal.ID, kind: storage.AddressBooks, object: objectHome}, principal.ID),
		)
	case levelHome:
		collections, err := s.store.ListCollections(principal.ID, target.kind)
		if err != nil {
			return nil, err
		}
		for _, coll := range collections {
			responses = append(responses, propertyResponseFor(davTarget{
				principal:  principal.ID,
				kind:       target.kind,
				collection: coll.Name,
				object:     objectCollection,
			}, principal.ID))
		}
	case levelCollection:
		resources, err := s.store.ListResources(principal.ID, target.kind, target.collection)
		if err != nil {
			return nil, err
		}
		for _, resource := range resources {
			responses = append(responses, propertyResponse{
				Href:         target.href() + url.PathEscape(resource.Name),
				ResourceType: "resource",
				DisplayName:  resource.Name,
				ETag:         resource.ETag,
				ContentType:  resource.ContentType,
			})
		}
	}
	return responses, nil
}

func propertyResponseFor(target davTarget, principalID string) propertyResponse {
	p := propertyResponse{
		Href:                 target.href(),
		DisplayName:          displayName(target),
		ResourceType:         target.resourceType(),
		CurrentUserPrincipal: "/dav/principals/" + url.PathEscape(principalID) + "/",
	}
	if target.object == objectPrincipal || target.level() == levelDAVRoot {
		p.CalendarHome = "/dav/calendars/" + url.PathEscape(principalID) + "/"
		p.AddressBookHome = "/dav/addressbooks/" + url.PathEscape(principalID) + "/"
	}
	return p
}

func parseReport(data []byte) (string, []string, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var root string
	var hrefs []string
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			if root == "" {
				root = t.Name.Local
				continue
			}
			if t.Name.Local == "href" {
				var href string
				if err := decoder.DecodeElement(&href, &t); err != nil {
					return "", nil, err
				}
				hrefs = append(hrefs, strings.TrimSpace(href))
			}
		}
	}
	switch root {
	case "calendar-query", "calendar-multiget", "addressbook-query", "addressbook-multiget":
		return root, hrefs, nil
	default:
		return "", nil, fmt.Errorf("unsupported report %q", root)
	}
}

func reportAllowed(name string, kind storage.Kind) bool {
	if kind == storage.Calendars {
		return name == "calendar-query" || name == "calendar-multiget"
	}
	return name == "addressbook-query" || name == "addressbook-multiget"
}

func validateResource(kind storage.Kind, data []byte) error {
	upper := strings.ToUpper(string(data))
	switch kind {
	case storage.Calendars:
		if !strings.Contains(upper, "BEGIN:VCALENDAR") || !strings.Contains(upper, "END:VCALENDAR") {
			return fmt.Errorf("calendar resource must contain a VCALENDAR object")
		}
	case storage.AddressBooks:
		if !strings.Contains(upper, "BEGIN:VCARD") || !strings.Contains(upper, "END:VCARD") {
			return fmt.Errorf("address-book resource must contain a VCARD object")
		}
	default:
		return fmt.Errorf("unsupported DAV resource kind")
	}
	return nil
}

func preconditionsSatisfied(r *http.Request, currentETag string, exists bool) bool {
	if noneMatch := strings.TrimSpace(r.Header.Get("If-None-Match")); noneMatch != "" {
		if noneMatch == "*" && exists {
			return false
		}
		if exists && tagListContains(noneMatch, currentETag) {
			return false
		}
	}
	if match := strings.TrimSpace(r.Header.Get("If-Match")); match != "" {
		if !exists {
			return false
		}
		if match != "*" && !tagListContains(match, currentETag) {
			return false
		}
	}
	return true
}

func tagListContains(raw, tag string) bool {
	for _, part := range strings.Split(raw, ",") {
		if strings.TrimSpace(part) == tag {
			return true
		}
	}
	return false
}

func writeStorageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, storage.ErrAlreadyExists):
		http.Error(w, "already exists", http.StatusMethodNotAllowed)
	case errors.Is(err, storage.ErrInvalidSegment):
		http.Error(w, "invalid DAV path", http.StatusBadRequest)
	case errors.Is(err, storage.ErrNotEmpty):
		http.Error(w, "collection is not empty", http.StatusConflict)
	default:
		http.Error(w, "storage operation failed", http.StatusInternalServerError)
	}
}

func allowedMethods(target davTarget) string {
	if target.resource != "" {
		return "OPTIONS, PROPFIND, GET, HEAD, PUT, DELETE"
	}
	if target.kind == storage.Calendars && target.collection != "" {
		return "OPTIONS, PROPFIND, MKCALENDAR, DELETE, REPORT"
	}
	if target.kind == storage.AddressBooks && target.collection != "" {
		return "OPTIONS, PROPFIND, MKCOL, DELETE, REPORT"
	}
	return "OPTIONS, PROPFIND"
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
