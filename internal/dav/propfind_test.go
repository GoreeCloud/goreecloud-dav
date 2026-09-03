package dav

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"strings"
	"testing"
)

func seedPropfindCalendarResource(t *testing.T, baseURL string, client *http.Client) string {
	t.Helper()
	resp := request(t, client, "MKCALENDAR", baseURL+"/dav/calendars/alice/personal/", "", nil)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("MKCALENDAR status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//GoreeCloud//PROPFIND Test//EN\r\nEND:VCALENDAR\r\n"
	resp = request(t, client, http.MethodPut, baseURL+"/dav/calendars/alice/personal/event.ics", ics, nil)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("PUT status=%d", resp.StatusCode)
	}
	etag := resp.Header.Get("ETag")
	resp.Body.Close()
	return etag
}

func xmlElementValues(t *testing.T, body []byte, namespace, local string) []string {
	t.Helper()
	decoder := xml.NewDecoder(bytes.NewReader(body))
	var values []string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return values
		}
		if err != nil {
			t.Fatalf("decode XML response: %v\n%s", err, body)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Space != namespace || start.Name.Local != local {
			continue
		}
		var value string
		if err := decoder.DecodeElement(&value, &start); err != nil {
			t.Fatalf("decode %s element: %v", local, err)
		}
		values = append(values, value)
	}
}

func TestPropfindEmptyBodyUsesAllPropBaseline(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedPropfindCalendarResource(t, ts.URL, client)

	resp := request(t, client, "PROPFIND", ts.URL+"/dav/calendars/alice/personal/event.ics", "", map[string]string{"Depth": "0"})
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 {
		t.Fatalf("PROPFIND empty body status=%d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(text, "getetag") || !strings.Contains(text, "getcontenttype") || !strings.Contains(text, "resourcetype") {
		t.Fatalf("allprop baseline omitted supported live properties: %s", body)
	}
}

func TestPropfindExplicitPropertiesSeparateMissingPropstat(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	etag := seedPropfindCalendarResource(t, ts.URL, client)

	bodyXML := `<D:propfind xmlns:D="DAV:" xmlns:X="urn:example:test"><D:prop><D:getetag/><X:unknown/></D:prop></D:propfind>`
	resp := request(t, client, "PROPFIND", ts.URL+"/dav/calendars/alice/personal/event.ics", bodyXML, map[string]string{"Depth": "0"})
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 {
		t.Fatalf("explicit PROPFIND status=%d body=%s", resp.StatusCode, body)
	}
	etags := xmlElementValues(t, body, nsDAV, "getetag")
	if len(etags) != 1 || etags[0] != etag || !strings.Contains(text, "unknown") || !strings.Contains(text, "HTTP/1.1 404 Not Found") {
		t.Fatalf("explicit PROPFIND did not separate supported/missing properties: %s", body)
	}
}

func TestPropfindPropnameReturnsNamesWithoutLiveValues(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedPropfindCalendarResource(t, ts.URL, client)

	bodyXML := `<D:propfind xmlns:D="DAV:"><D:propname/></D:propfind>`
	resp := request(t, client, "PROPFIND", ts.URL+"/dav/calendars/alice/personal/event.ics", bodyXML, map[string]string{"Depth": "0"})
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 207 {
		t.Fatalf("propname PROPFIND status=%d body=%s", resp.StatusCode, body)
	}
	etags := xmlElementValues(t, body, nsDAV, "getetag")
	if len(etags) != 1 || etags[0] != "" {
		t.Fatalf("propname should expose property names without live values: %s", body)
	}
}

func TestPropfindRejectsMalformedSelection(t *testing.T) {
	ts := newTestServer(t)
	bodyXML := `<D:propfind xmlns:D="DAV:"><D:allprop/><D:propname/></D:propfind>`
	resp := request(t, ts.Client(), "PROPFIND", ts.URL+"/dav/", bodyXML, map[string]string{"Depth": "0"})
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed PROPFIND selection status=%d", resp.StatusCode)
	}
}

func TestPropfindExplicitUnavailableLivePropertyReturns404Propstat(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedPropfindCalendarResource(t, ts.URL, client)

	bodyXML := `<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav"><D:prop><C:calendar-home-set/></D:prop></D:propfind>`
	resp := request(t, client, "PROPFIND", ts.URL+"/dav/calendars/alice/personal/event.ics", bodyXML, map[string]string{"Depth": "0"})
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 207 || !strings.Contains(string(body), "HTTP/1.1 404 Not Found") {
		t.Fatalf("unavailable live property status=%d body=%s", resp.StatusCode, body)
	}
}

func TestPropfindMissingDepthUsesInfinitySemanticsAndFailsFinite(t *testing.T) {
	ts := newTestServer(t)
	bodyXML := `<D:propfind xmlns:D="DAV:"><D:allprop/></D:propfind>`
	resp := request(t, ts.Client(), "PROPFIND", ts.URL+"/dav/", bodyXML, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("missing Depth status=%d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "propfind-finite-depth") {
		t.Fatalf("missing Depth did not return DAV finite-depth precondition: %s", body)
	}
}

func TestPropfindInfinityReturnsFiniteDepthPrecondition(t *testing.T) {
	ts := newTestServer(t)
	bodyXML := `<D:propfind xmlns:D="DAV:"><D:allprop/></D:propfind>`
	resp := request(t, ts.Client(), "PROPFIND", ts.URL+"/dav/", bodyXML, map[string]string{"Depth": "infinity"})
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden || !strings.Contains(string(body), "propfind-finite-depth") {
		t.Fatalf("infinite Depth status=%d body=%s", resp.StatusCode, body)
	}
}

func TestPropfindRejectsInvalidDepthValue(t *testing.T) {
	ts := newTestServer(t)
	bodyXML := `<D:propfind xmlns:D="DAV:"><D:allprop/></D:propfind>`
	resp := request(t, ts.Client(), "PROPFIND", ts.URL+"/dav/", bodyXML, map[string]string{"Depth": "2"})
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid Depth status=%d", resp.StatusCode)
	}
}
