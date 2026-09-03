package dav

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func seedCalendarResource(t *testing.T, baseURL string, client *http.Client) {
	t.Helper()

	resp := request(t, client, "MKCALENDAR", baseURL+"/dav/calendars/alice/personal/", "", nil)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("MKCALENDAR status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//GoreeCloud//DAV Report Test//EN\r\nEND:VCALENDAR\r\n"
	resp = request(t, client, http.MethodPut, baseURL+"/dav/calendars/alice/personal/event.ics", ics, nil)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("PUT status=%d", resp.StatusCode)
	}
	resp.Body.Close()
}

func seedAddressBookResource(t *testing.T, baseURL string, client *http.Client) {
	t.Helper()

	resp := request(t, client, "MKCOL", baseURL+"/dav/addressbooks/alice/contacts/", "", nil)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("MKCOL status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	vcard := "BEGIN:VCARD\r\nVERSION:4.0\r\nFN:Report Test\r\nEND:VCARD\r\n"
	resp = request(t, client, http.MethodPut, baseURL+"/dav/addressbooks/alice/contacts/person.vcf", vcard, nil)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("vCard PUT status=%d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestCalendarMultigetAcceptsInScopeDAVHref(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarResource(t, ts.URL, client)

	report := `<C:calendar-multiget xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:"><D:href>/dav/calendars/alice/personal/event.ics</D:href></C:calendar-multiget>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 207 || !strings.Contains(string(body), "VCALENDAR") {
		t.Fatalf("in-scope multiget status=%d body=%s", resp.StatusCode, body)
	}
}

func TestCalendarMultigetReturnsPerHrefNotFound(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarResource(t, ts.URL, client)

	report := `<C:calendar-multiget xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:"><D:href>/dav/calendars/alice/personal/event.ics</D:href><D:href>/dav/calendars/alice/personal/missing.ics</D:href></C:calendar-multiget>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 {
		t.Fatalf("multiget status=%d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(text, "event.ics") || !strings.Contains(text, "missing.ics") || !strings.Contains(text, "HTTP/1.1 404 Not Found") {
		t.Fatalf("multiget must return one response per requested href: %s", body)
	}
}

func TestAddressBookMultigetReturnsPerHrefNotFound(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedAddressBookResource(t, ts.URL, client)

	report := `<C:addressbook-multiget xmlns:C="urn:ietf:params:xml:ns:carddav" xmlns:D="DAV:"><D:href>/dav/addressbooks/alice/contacts/person.vcf</D:href><D:href>/dav/addressbooks/alice/contacts/missing.vcf</D:href></C:addressbook-multiget>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/addressbooks/alice/contacts/", report, map[string]string{"Depth": "0"})
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 {
		t.Fatalf("addressbook multiget status=%d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(text, "person.vcf") || !strings.Contains(text, "missing.vcf") || !strings.Contains(text, "HTTP/1.1 404 Not Found") {
		t.Fatalf("addressbook multiget must return one response per requested href: %s", body)
	}
}

func TestCalendarMultigetRejectsOutOfScopeDAVHref(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarResource(t, ts.URL, client)

	report := `<C:calendar-multiget xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:"><D:href>/dav/calendars/bob/personal/event.ics</D:href></C:calendar-multiget>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("out-of-scope multiget status=%d body=%s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), "VCALENDAR") {
		t.Fatalf("out-of-scope multiget disclosed local resource data: %s", body)
	}
}

func TestCalendarMultigetRequiresDAVHref(t *testing.T) {
	ts := newTestServer(t)
	report := `<C:calendar-multiget xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:"/>`
	resp := request(t, ts.Client(), "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("multiget without href status=%d", resp.StatusCode)
	}
}

func TestReportRejectsUnexpectedNamespace(t *testing.T) {
	ts := newTestServer(t)
	report := `<calendar-query xmlns="urn:example:wrong"/>`
	resp := request(t, ts.Client(), "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("wrong-namespace report status=%d", resp.StatusCode)
	}
}
