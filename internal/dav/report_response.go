package dav

import (
	"encoding/xml"
	"net/http"

	"github.com/GoreeCloud/goreecloud-dav/internal/storage"
)

func writeReportMultiStatus(w http.ResponseWriter, responses []propertyResponse, request reportRequest) {
	w.Header().Set("Content-Type", `application/xml; charset="utf-8"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(207)

	enc := xml.NewEncoder(w)
	_ = enc.EncodeToken(xml.ProcInst{Target: "xml", Inst: []byte(`version="1.0" encoding="UTF-8"`)})
	root := xml.StartElement{Name: xml.Name{Space: nsDAV, Local: "multistatus"}}
	_ = enc.EncodeToken(root)
	for _, response := range responses {
		encodeReportResponse(enc, response, request)
	}
	_ = enc.EncodeToken(root.End())
	_ = enc.Flush()
}

func encodeReportResponse(enc *xml.Encoder, response propertyResponse, request reportRequest) {
	start := xml.StartElement{Name: xml.Name{Space: nsDAV, Local: "response"}}
	_ = enc.EncodeToken(start)
	encodeText(enc, nsDAV, "href", response.Href)

	if response.Status != "" {
		encodeText(enc, nsDAV, "status", response.Status)
		_ = enc.EncodeToken(start.End())
		return
	}

	if request.PropertyMode == reportPropertiesNone {
		encodeText(enc, nsDAV, "status", "HTTP/1.1 200 OK")
		_ = enc.EncodeToken(start.End())
		return
	}

	available := availableProperties(response)
	availableByName := make(map[xml.Name]availableProperty, len(available)+2)
	for _, property := range available {
		availableByName[property.Name] = property
	}
	if response.Data != "" {
		data := response.Data
		switch response.DataKind {
		case storage.Calendars:
			availableByName[xml.Name{Space: nsCalDAV, Local: "calendar-data"}] = availableProperty{
				Name: xml.Name{Space: nsCalDAV, Local: "calendar-data"},
				Encode: func(enc *xml.Encoder) {
					encodeText(enc, nsCalDAV, "calendar-data", data)
				},
			}
		case storage.AddressBooks:
			availableByName[xml.Name{Space: nsCardDAV, Local: "address-data"}] = availableProperty{
				Name: xml.Name{Space: nsCardDAV, Local: "address-data"},
				Encode: func(enc *xml.Encoder) {
					encodeText(enc, nsCardDAV, "address-data", data)
				},
			}
		}
	}

	var successful []availableProperty
	var missing []xml.Name
	switch request.PropertyMode {
	case reportPropertiesAll, reportPropertyNames:
		// DAV:allprop and DAV:propname cover WebDAV live properties. The
		// CalDAV/CardDAV data elements are report-specific selectors rather
		// than ordinary live properties and are returned only when explicitly
		// requested in DAV:prop.
		successful = available
	case reportPropertiesExplicit:
		for _, name := range request.Properties {
			property, ok := availableByName[name]
			if !ok {
				missing = append(missing, name)
				continue
			}
			successful = append(successful, property)
		}
	}

	if len(successful) > 0 {
		encodePropfindPropstat(enc, successful, nil, request.PropertyMode == reportPropertyNames, "HTTP/1.1 200 OK")
	}
	if len(missing) > 0 {
		encodePropfindPropstat(enc, nil, missing, true, "HTTP/1.1 404 Not Found")
	}
	if len(successful) == 0 && len(missing) == 0 {
		encodeText(enc, nsDAV, "status", "HTTP/1.1 200 OK")
	}
	_ = enc.EncodeToken(start.End())
}
