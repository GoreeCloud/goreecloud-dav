package dav

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCalendarMultigetETagOnlyDoesNotReturnCalendarData(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarResource(t, ts.URL, client)

	report := `<C:calendar-multiget xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:"><D:prop><D:getetag/></D:prop><D:href>/dav/calendars/alice/personal/event.ics</D:href></C:calendar-multiget>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 || !strings.Contains(text, "getetag") {
		t.Fatalf("etag-only multiget status=%d body=%s", resp.StatusCode, body)
	}
	if strings.Contains(text, "VCALENDAR") || strings.Contains(text, "calendar-data") {
		t.Fatalf("etag-only multiget leaked calendar data: %s", body)
	}
}

func TestCalendarMultigetExplicitCalendarDataReturnsPayload(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarResource(t, ts.URL, client)

	report := `<C:calendar-multiget xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:"><D:prop><C:calendar-data/></D:prop><D:href>/dav/calendars/alice/personal/event.ics</D:href></C:calendar-multiget>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 || !strings.Contains(text, "calendar-data") || !strings.Contains(text, "VCALENDAR") {
		t.Fatalf("calendar-data multiget status=%d body=%s", resp.StatusCode, body)
	}
	if strings.Contains(text, "getetag") {
		t.Fatalf("calendar-data-only multiget returned unrequested ETag: %s", body)
	}
}

func TestCalendarMultigetUnknownPropertyGets404Propstat(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarResource(t, ts.URL, client)

	report := `<C:calendar-multiget xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:" xmlns:X="urn:example:test"><D:prop><D:getetag/><X:unknown/></D:prop><D:href>/dav/calendars/alice/personal/event.ics</D:href></C:calendar-multiget>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 || !strings.Contains(text, "getetag") || !strings.Contains(text, "unknown") || !strings.Contains(text, "HTTP/1.1 404 Not Found") {
		t.Fatalf("mixed report property status=%d body=%s", resp.StatusCode, body)
	}
}

func TestCalendarMultigetRejectsUnsupportedPartialCalendarProjection(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarResource(t, ts.URL, client)

	report := `<C:calendar-multiget xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:"><D:prop><C:calendar-data><C:comp name="VCALENDAR"/></C:calendar-data></D:prop><D:href>/dav/calendars/alice/personal/event.ics</D:href></C:calendar-multiget>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("partial calendar projection status=%d body=%s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), "VCALENDAR") {
		t.Fatalf("unsupported partial calendar projection leaked payload: %s", body)
	}
}

func TestAddressBookMultigetETagOnlyDoesNotReturnVCardData(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedAddressBookResource(t, ts.URL, client)

	report := `<C:addressbook-multiget xmlns:C="urn:ietf:params:xml:ns:carddav" xmlns:D="DAV:"><D:prop><D:getetag/></D:prop><D:href>/dav/addressbooks/alice/contacts/person.vcf</D:href></C:addressbook-multiget>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/addressbooks/alice/contacts/", report, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 || !strings.Contains(text, "getetag") {
		t.Fatalf("etag-only addressbook multiget status=%d body=%s", resp.StatusCode, body)
	}
	if strings.Contains(text, "VCARD") || strings.Contains(text, "address-data") {
		t.Fatalf("etag-only addressbook multiget leaked vCard data: %s", body)
	}
}

func TestAddressBookMultigetExplicitAddressDataReturnsPayload(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedAddressBookResource(t, ts.URL, client)

	report := `<C:addressbook-multiget xmlns:C="urn:ietf:params:xml:ns:carddav" xmlns:D="DAV:"><D:prop><C:address-data/></D:prop><D:href>/dav/addressbooks/alice/contacts/person.vcf</D:href></C:addressbook-multiget>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/addressbooks/alice/contacts/", report, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 || !strings.Contains(text, "address-data") || !strings.Contains(text, "VCARD") {
		t.Fatalf("address-data multiget status=%d body=%s", resp.StatusCode, body)
	}
}

func TestReportWithoutPropertySelectionDoesNotReturnObjectData(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarResource(t, ts.URL, client)

	report := `<C:calendar-multiget xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:"><D:href>/dav/calendars/alice/personal/event.ics</D:href></C:calendar-multiget>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 || !strings.Contains(text, "event.ics") || !strings.Contains(text, "HTTP/1.1 200 OK") {
		t.Fatalf("propertyless multiget status=%d body=%s", resp.StatusCode, body)
	}
	if strings.Contains(text, "VCALENDAR") || strings.Contains(text, "getetag") {
		t.Fatalf("propertyless multiget returned unrequested resource properties/data: %s", body)
	}
}

func TestReportAllpropDoesNotImplicitlyReturnCalendarData(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarResource(t, ts.URL, client)

	report := `<C:calendar-multiget xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:"><D:allprop/><D:href>/dav/calendars/alice/personal/event.ics</D:href></C:calendar-multiget>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 || !strings.Contains(text, "getetag") || !strings.Contains(text, "getcontenttype") {
		t.Fatalf("allprop multiget status=%d body=%s", resp.StatusCode, body)
	}
	if strings.Contains(text, "VCALENDAR") || strings.Contains(text, "calendar-data") {
		t.Fatalf("DAV:allprop implicitly returned report-specific calendar data: %s", body)
	}
}
