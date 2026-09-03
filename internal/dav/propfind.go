package dav

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/GoreeCloud/goreecloud-dav/internal/storage"
)

type propfindMode uint8

const (
	propfindAllProp propfindMode = iota
	propfindPropName
	propfindProp
)

type propfindRequest struct {
	Mode  propfindMode
	Props []xml.Name
}

type availableProperty struct {
	Name   xml.Name
	Encode func(*xml.Encoder)
}

func parsePropfind(data []byte) (propfindRequest, error) {
	if strings.TrimSpace(string(data)) == "" {
		return propfindRequest{Mode: propfindAllProp}, nil
	}

	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	depth := 0
	var root xml.Name
	var request propfindRequest
	modeSet := false
	inProp := false

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return propfindRequest{}, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				root = t.Name
				if root.Space != nsDAV || root.Local != "propfind" {
					return propfindRequest{}, fmt.Errorf("PROPFIND body root must be DAV:propfind")
				}
				depth++
				continue
			}

			if depth == 1 {
				if modeSet {
					return propfindRequest{}, fmt.Errorf("PROPFIND body must contain exactly one property selection")
				}
				if t.Name.Space != nsDAV {
					return propfindRequest{}, fmt.Errorf("PROPFIND selection must use DAV namespace")
				}
				switch t.Name.Local {
				case "allprop":
					request.Mode = propfindAllProp
					modeSet = true
				case "propname":
					request.Mode = propfindPropName
					modeSet = true
				case "prop":
					request.Mode = propfindProp
					modeSet = true
					inProp = true
				default:
					return propfindRequest{}, fmt.Errorf("unsupported PROPFIND selection %q", t.Name.Local)
				}
			} else if inProp && depth == 2 {
				request.Props = append(request.Props, t.Name)
			}
			depth++

		case xml.EndElement:
			depth--
			if depth < 0 {
				return propfindRequest{}, fmt.Errorf("malformed PROPFIND body")
			}
			if inProp && depth == 1 && t.Name.Space == nsDAV && t.Name.Local == "prop" {
				inProp = false
			}
		}
	}

	if root.Local == "" || depth != 0 || !modeSet {
		return propfindRequest{}, fmt.Errorf("incomplete PROPFIND body")
	}
	if request.Mode == propfindProp && len(request.Props) == 0 {
		return propfindRequest{}, fmt.Errorf("DAV:prop must request at least one property")
	}
	request.Props = deduplicatePropertyNames(request.Props)
	return request, nil
}

func deduplicatePropertyNames(names []xml.Name) []xml.Name {
	seen := make(map[xml.Name]struct{}, len(names))
	out := make([]xml.Name, 0, len(names))
	for _, name := range names {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func writePropfindFiniteDepthError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", `application/xml; charset="utf-8"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusForbidden)

	enc := xml.NewEncoder(w)
	_ = enc.EncodeToken(xml.ProcInst{Target: "xml", Inst: []byte(`version="1.0" encoding="UTF-8"`)})
	root := xml.StartElement{Name: xml.Name{Space: nsDAV, Local: "error"}}
	_ = enc.EncodeToken(root)
	encodeEmpty(enc, nsDAV, "propfind-finite-depth")
	_ = enc.EncodeToken(root.End())
	_ = enc.Flush()
}

func writePropfindMultiStatus(w http.ResponseWriter, responses []propertyResponse, request propfindRequest) {
	w.Header().Set("Content-Type", `application/xml; charset="utf-8"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(207)

	enc := xml.NewEncoder(w)
	_ = enc.EncodeToken(xml.ProcInst{Target: "xml", Inst: []byte(`version="1.0" encoding="UTF-8"`)})
	root := xml.StartElement{Name: xml.Name{Space: nsDAV, Local: "multistatus"}}
	_ = enc.EncodeToken(root)
	for _, response := range responses {
		encodePropfindResponse(enc, response, request)
	}
	_ = enc.EncodeToken(root.End())
	_ = enc.Flush()
}

func encodePropfindResponse(enc *xml.Encoder, response propertyResponse, request propfindRequest) {
	start := xml.StartElement{Name: xml.Name{Space: nsDAV, Local: "response"}}
	_ = enc.EncodeToken(start)
	encodeText(enc, nsDAV, "href", response.Href)

	available := availableProperties(response)
	availableByName := make(map[xml.Name]availableProperty, len(available))
	for _, property := range available {
		availableByName[property.Name] = property
	}

	var successful []availableProperty
	var missing []xml.Name
	switch request.Mode {
	case propfindAllProp, propfindPropName:
		successful = available
	case propfindProp:
		for _, name := range request.Props {
			property, ok := availableByName[name]
			if !ok {
				missing = append(missing, name)
				continue
			}
			successful = append(successful, property)
		}
	}

	if len(successful) > 0 {
		encodePropfindPropstat(enc, successful, nil, request.Mode == propfindPropName, "HTTP/1.1 200 OK")
	}
	if len(missing) > 0 {
		encodePropfindPropstat(enc, nil, missing, true, "HTTP/1.1 404 Not Found")
	}
	_ = enc.EncodeToken(start.End())
}

func encodePropfindPropstat(enc *xml.Encoder, properties []availableProperty, names []xml.Name, namesOnly bool, status string) {
	propstat := xml.StartElement{Name: xml.Name{Space: nsDAV, Local: "propstat"}}
	_ = enc.EncodeToken(propstat)
	prop := xml.StartElement{Name: xml.Name{Space: nsDAV, Local: "prop"}}
	_ = enc.EncodeToken(prop)
	for _, property := range properties {
		if namesOnly {
			encodeEmpty(enc, property.Name.Space, property.Name.Local)
		} else {
			property.Encode(enc)
		}
	}
	for _, name := range names {
		encodeEmpty(enc, name.Space, name.Local)
	}
	_ = enc.EncodeToken(prop.End())
	encodeText(enc, nsDAV, "status", status)
	_ = enc.EncodeToken(propstat.End())
}

func availableProperties(response propertyResponse) []availableProperty {
	properties := make([]availableProperty, 0, 7)
	if response.DisplayName != "" {
		value := response.DisplayName
		properties = append(properties, availableProperty{
			Name: xml.Name{Space: nsDAV, Local: "displayname"},
			Encode: func(enc *xml.Encoder) {
				encodeText(enc, nsDAV, "displayname", value)
			},
		})
	}

	resourceType := response.ResourceType
	properties = append(properties, availableProperty{
		Name: xml.Name{Space: nsDAV, Local: "resourcetype"},
		Encode: func(enc *xml.Encoder) {
			encodeResourceType(enc, resourceType)
		},
	})

	if response.CurrentUserPrincipal != "" {
		value := response.CurrentUserPrincipal
		properties = append(properties, availableProperty{
			Name: xml.Name{Space: nsDAV, Local: "current-user-principal"},
			Encode: func(enc *xml.Encoder) {
				encodeHrefProperty(enc, nsDAV, "current-user-principal", value)
			},
		})
	}
	if response.CalendarHome != "" {
		value := response.CalendarHome
		properties = append(properties, availableProperty{
			Name: xml.Name{Space: nsCalDAV, Local: "calendar-home-set"},
			Encode: func(enc *xml.Encoder) {
				encodeHrefProperty(enc, nsCalDAV, "calendar-home-set", value)
			},
		})
	}
	if response.AddressBookHome != "" {
		value := response.AddressBookHome
		properties = append(properties, availableProperty{
			Name: xml.Name{Space: nsCardDAV, Local: "addressbook-home-set"},
			Encode: func(enc *xml.Encoder) {
				encodeHrefProperty(enc, nsCardDAV, "addressbook-home-set", value)
			},
		})
	}
	if response.ETag != "" {
		value := response.ETag
		properties = append(properties, availableProperty{
			Name: xml.Name{Space: nsDAV, Local: "getetag"},
			Encode: func(enc *xml.Encoder) {
				encodeText(enc, nsDAV, "getetag", value)
			},
		})
	}
	if response.ContentType != "" {
		value := response.ContentType
		properties = append(properties, availableProperty{
			Name: xml.Name{Space: nsDAV, Local: "getcontenttype"},
			Encode: func(enc *xml.Encoder) {
				encodeText(enc, nsDAV, "getcontenttype", value)
			},
		})
	}
	return properties
}

func propfindBodyReadError(err error) int {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}

func isSupportedLiveProperty(name xml.Name) bool {
	switch name {
	case xml.Name{Space: nsDAV, Local: "displayname"},
		xml.Name{Space: nsDAV, Local: "resourcetype"},
		xml.Name{Space: nsDAV, Local: "current-user-principal"},
		xml.Name{Space: nsDAV, Local: "getetag"},
		xml.Name{Space: nsDAV, Local: "getcontenttype"},
		xml.Name{Space: nsCalDAV, Local: "calendar-home-set"},
		xml.Name{Space: nsCardDAV, Local: "addressbook-home-set"}:
		return true
	default:
		return false
	}
}

func propertyKindForName(name xml.Name) storage.Kind {
	switch name.Space {
	case nsCalDAV:
		return storage.Calendars
	case nsCardDAV:
		return storage.AddressBooks
	default:
		return ""
	}
}
