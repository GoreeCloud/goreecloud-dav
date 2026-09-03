package dav

import (
	"encoding/xml"
	"net/http"

	"github.com/GoreeCloud/goreecloud-dav/internal/storage"
)

const (
	nsDAV     = "DAV:"
	nsCalDAV  = "urn:ietf:params:xml:ns:caldav"
	nsCardDAV = "urn:ietf:params:xml:ns:carddav"
)

type propertyResponse struct {
	Href                 string
	Status               string
	DisplayName          string
	ResourceType         string
	CurrentUserPrincipal string
	CalendarHome         string
	AddressBookHome      string
	ETag                 string
	ContentType          string
	Data                 string
	DataKind             storage.Kind
}

func writeMultiStatus(w http.ResponseWriter, responses []propertyResponse) {
	w.Header().Set("Content-Type", `application/xml; charset="utf-8"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(207)

	enc := xml.NewEncoder(w)
	_ = enc.EncodeToken(xml.ProcInst{Target: "xml", Inst: []byte(`version="1.0" encoding="UTF-8"`)})
	root := xml.StartElement{Name: xml.Name{Space: nsDAV, Local: "multistatus"}}
	_ = enc.EncodeToken(root)
	for _, response := range responses {
		encodeResponse(enc, response)
	}
	_ = enc.EncodeToken(root.End())
	_ = enc.Flush()
}

func encodeResponse(enc *xml.Encoder, p propertyResponse) {
	response := xml.StartElement{Name: xml.Name{Space: nsDAV, Local: "response"}}
	_ = enc.EncodeToken(response)
	encodeText(enc, nsDAV, "href", p.Href)
	if p.Status != "" {
		encodeText(enc, nsDAV, "status", p.Status)
		_ = enc.EncodeToken(response.End())
		return
	}

	propstat := xml.StartElement{Name: xml.Name{Space: nsDAV, Local: "propstat"}}
	_ = enc.EncodeToken(propstat)
	prop := xml.StartElement{Name: xml.Name{Space: nsDAV, Local: "prop"}}
	_ = enc.EncodeToken(prop)

	if p.DisplayName != "" {
		encodeText(enc, nsDAV, "displayname", p.DisplayName)
	}
	encodeResourceType(enc, p.ResourceType)
	if p.CurrentUserPrincipal != "" {
		encodeHrefProperty(enc, nsDAV, "current-user-principal", p.CurrentUserPrincipal)
	}
	if p.CalendarHome != "" {
		encodeHrefProperty(enc, nsCalDAV, "calendar-home-set", p.CalendarHome)
	}
	if p.AddressBookHome != "" {
		encodeHrefProperty(enc, nsCardDAV, "addressbook-home-set", p.AddressBookHome)
	}
	if p.ETag != "" {
		encodeText(enc, nsDAV, "getetag", p.ETag)
	}
	if p.ContentType != "" {
		encodeText(enc, nsDAV, "getcontenttype", p.ContentType)
	}
	if p.Data != "" {
		if p.DataKind == storage.Calendars {
			encodeText(enc, nsCalDAV, "calendar-data", p.Data)
		} else if p.DataKind == storage.AddressBooks {
			encodeText(enc, nsCardDAV, "address-data", p.Data)
		}
	}

	_ = enc.EncodeToken(prop.End())
	encodeText(enc, nsDAV, "status", "HTTP/1.1 200 OK")
	_ = enc.EncodeToken(propstat.End())
	_ = enc.EncodeToken(response.End())
}

func encodeResourceType(enc *xml.Encoder, kind string) {
	start := xml.StartElement{Name: xml.Name{Space: nsDAV, Local: "resourcetype"}}
	_ = enc.EncodeToken(start)
	switch kind {
	case "principal":
		encodeEmpty(enc, nsDAV, "principal")
	case "collection":
		encodeEmpty(enc, nsDAV, "collection")
	case "calendar":
		encodeEmpty(enc, nsDAV, "collection")
		encodeEmpty(enc, nsCalDAV, "calendar")
	case "addressbook":
		encodeEmpty(enc, nsDAV, "collection")
		encodeEmpty(enc, nsCardDAV, "addressbook")
	}
	_ = enc.EncodeToken(start.End())
}

func encodeHrefProperty(enc *xml.Encoder, namespace, local, href string) {
	start := xml.StartElement{Name: xml.Name{Space: namespace, Local: local}}
	_ = enc.EncodeToken(start)
	encodeText(enc, nsDAV, "href", href)
	_ = enc.EncodeToken(start.End())
}

func encodeText(enc *xml.Encoder, namespace, local, value string) {
	start := xml.StartElement{Name: xml.Name{Space: namespace, Local: local}}
	_ = enc.EncodeToken(start)
	_ = enc.EncodeToken(xml.CharData([]byte(value)))
	_ = enc.EncodeToken(start.End())
}

func encodeEmpty(enc *xml.Encoder, namespace, local string) {
	start := xml.StartElement{Name: xml.Name{Space: namespace, Local: local}}
	_ = enc.EncodeToken(start)
	_ = enc.EncodeToken(start.End())
}
