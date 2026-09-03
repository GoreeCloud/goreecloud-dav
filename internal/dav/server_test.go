package dav

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoreeCloud/goreecloud-dav/internal/auth"
	"github.com/GoreeCloud/goreecloud-dav/internal/storage"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	store, err := storage.NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := New(store, auth.StaticProvider{PrincipalID: "alice"}, 1024*1024)
	ts := httptest.NewServer(server.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func request(t *testing.T, client *http.Client, method, url, body string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestDAVCalendarLifecycle(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()

	resp := request(t, client, "OPTIONS", ts.URL+"/dav/", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("OPTIONS status=%d", resp.StatusCode)
	}
	if got := resp.Header.Get("DAV"); got != "" {
		t.Fatalf("foundation must not advertise an unearned DAV compliance class: %q", got)
	}
	if !strings.Contains(resp.Header.Get("Allow"), "PROPFIND") {
		t.Fatalf("OPTIONS did not expose implemented methods: %q", resp.Header.Get("Allow"))
	}
	resp.Body.Close()

	resp = request(t, client, "MKCALENDAR", ts.URL+"/dav/calendars/alice/personal/", "", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("MKCALENDAR status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//GoreeCloud//DAV Test//EN\r\nBEGIN:VEVENT\r\nUID:event-1\r\nDTSTAMP:20260903T120000Z\r\nDTSTART:20260904T120000Z\r\nSUMMARY:Test Event\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	resp = request(t, client, http.MethodPut, ts.URL+"/dav/calendars/alice/personal/event.ics", ics, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("PUT status=%d", resp.StatusCode)
	}
	etag := resp.Header.Get("ETag")
	if etag == "" {
		t.Fatal("PUT did not return ETag")
	}
	resp.Body.Close()

	resp = request(t, client, http.MethodGet, ts.URL+"/dav/calendars/alice/personal/event.ics", "", nil)
	got, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(got) != ics {
		t.Fatalf("GET status=%d body=%q", resp.StatusCode, got)
	}

	resp = request(t, client, http.MethodPut, ts.URL+"/dav/calendars/alice/personal/event.ics", ics, map[string]string{"If-None-Match": "*"})
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("conditional PUT status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = request(t, client, "PROPFIND", ts.URL+"/dav/calendars/alice/personal/", "", map[string]string{"Depth": "1"})
	propBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 207 || !strings.Contains(string(propBody), "calendar") || !strings.Contains(string(propBody), "event.ics") {
		t.Fatalf("PROPFIND status=%d body=%s", resp.StatusCode, propBody)
	}

	report := `<C:calendar-query xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:"><C:filter><C:comp-filter name="VCALENDAR"><C:comp-filter name="VEVENT"/></C:comp-filter></C:filter></C:calendar-query>`
	resp = request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, map[string]string{"Depth": "1"})
	reportBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 207 || !strings.Contains(string(reportBody), "VCALENDAR") {
		t.Fatalf("REPORT status=%d body=%s", resp.StatusCode, reportBody)
	}
}

func TestDAVAddressBookAndPrincipalIsolation(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()

	resp := request(t, client, "MKCOL", ts.URL+"/dav/addressbooks/alice/contacts/", "", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("MKCOL status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	vcard := "BEGIN:VCARD\r\nVERSION:4.0\r\nFN:Example Person\r\nEND:VCARD\r\n"
	resp = request(t, client, http.MethodPut, ts.URL+"/dav/addressbooks/alice/contacts/person.vcf", vcard, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("vCard PUT status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = request(t, client, "PROPFIND", ts.URL+"/dav/addressbooks/bob/", "", map[string]string{"Depth": "0"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected principal isolation, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestStatusDoesNotClaimPlatformConformance(t *testing.T) {
	ts := newTestServer(t)
	resp := request(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/status", "", nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status endpoint=%d", resp.StatusCode)
	}
	text := string(body)
	for _, integration := range []string{"goreecloud_identity", "privacy_shield", "wardveil_security", "everkeep", "goreecloud_manager", "goreecloud_mesh"} {
		if !strings.Contains(text, `"`+integration+`":"migration-required"`) {
			t.Fatalf("status must report %s as migration-required: %s", integration, text)
		}
	}
}

func TestWellKnownRedirectAndNoFalseComplianceToken(t *testing.T) {
	ts := newTestServer(t)
	client := *ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp := request(t, &client, http.MethodGet, ts.URL+"/.well-known/caldav", "", nil)
	if resp.StatusCode != http.StatusTemporaryRedirect || resp.Header.Get("Location") != "/dav/" {
		t.Fatalf("well-known status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()

	resp = request(t, &client, "OPTIONS", ts.URL+"/dav/", "", nil)
	if got := resp.Header.Get("DAV"); got != "" {
		t.Fatalf("foundation must not advertise RFC 4918/CalDAV/CardDAV compliance: %q", got)
	}
	resp.Body.Close()
}

func TestPutRequiresExistingCollection(t *testing.T) {
	ts := newTestServer(t)
	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n"
	resp := request(t, ts.Client(), http.MethodPut, ts.URL+"/dav/calendars/alice/missing/event.ics", ics, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("PUT to missing collection status=%d", resp.StatusCode)
	}
	resp.Body.Close()
}
