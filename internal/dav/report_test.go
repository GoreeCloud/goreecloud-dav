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
