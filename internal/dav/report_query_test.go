package dav

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func seedCalendarQueryResources(t *testing.T, baseURL string, client *http.Client) {
	t.Helper()
	resp := request(t, client, "MKCALENDAR", baseURL+"/dav/calendars/alice/personal/", "", nil)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("MKCALENDAR status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	event := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//GoreeCloud//Query Test//EN\r\nBEGIN:VEVENT\r\nUID:event-query\r\nDTSTAMP:20260903T120000Z\r\nDTSTART:20260904T120000Z\r\nSUMMARY:Query Event\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	resp = request(t, client, http.MethodPut, baseURL+"/dav/calendars/alice/personal/event.ics", event, nil)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("event PUT status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	task := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//GoreeCloud//Query Test//EN\r\nBEGIN:VTODO\r\nUID:task-query\r\nDTSTAMP:20260903T120000Z\r\nSUMMARY:Query Task\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
	resp = request(t, client, http.MethodPut, baseURL+"/dav/calendars/alice/personal/task.ics", task, nil)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("task PUT status=%d", resp.StatusCode)
	}
	resp.Body.Close()
}

func seedAddressBookQueryResources(t *testing.T, baseURL string, client *http.Client) {
	t.Helper()
	resp := request(t, client, "MKCOL", baseURL+"/dav/addressbooks/alice/contacts/", "", nil)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("MKCOL status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	withEmail := "BEGIN:VCARD\r\nVERSION:4.0\r\nFN:Email Person\r\nitem1.EMAIL:person@example.test\r\nEND:VCARD\r\n"
	resp = request(t, client, http.MethodPut, baseURL+"/dav/addressbooks/alice/contacts/email.vcf", withEmail, nil)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("email vCard PUT status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	withoutEmail := "BEGIN:VCARD\r\nVERSION:4.0\r\nFN:No Email Person\r\nTEL:+15555550100\r\nEND:VCARD\r\n"
	resp = request(t, client, http.MethodPut, baseURL+"/dav/addressbooks/alice/contacts/phone.vcf", withoutEmail, nil)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("phone vCard PUT status=%d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestCalendarQueryRequiresFilter(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarQueryResources(t, ts.URL, client)

	report := `<C:calendar-query xmlns:C="urn:ietf:params:xml:ns:caldav"/>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, map[string]string{"Depth": "1"})
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("filterless calendar-query status=%d", resp.StatusCode)
	}
}

func TestCalendarQueryComponentExistenceFilter(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarQueryResources(t, ts.URL, client)

	report := `<C:calendar-query xmlns:C="urn:ietf:params:xml:ns:caldav"><C:filter><C:comp-filter name="VCALENDAR"><C:comp-filter name="VEVENT"/></C:comp-filter></C:filter></C:calendar-query>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, map[string]string{"Depth": "1"})
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 {
		t.Fatalf("calendar-query status=%d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(text, "event.ics") {
		t.Fatalf("VEVENT query omitted matching event href: %s", body)
	}
	if strings.Contains(text, "task.ics") {
		t.Fatalf("VEVENT query included non-matching task href: %s", body)
	}
	if strings.Contains(text, "VEVENT") || strings.Contains(text, "VTODO") {
		t.Fatalf("query without DAV property selection returned unrequested calendar data: %s", body)
	}
}

func TestCalendarQueryMissingDepthDefaultsToZero(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarQueryResources(t, ts.URL, client)

	report := `<C:calendar-query xmlns:C="urn:ietf:params:xml:ns:caldav"><C:filter><C:comp-filter name="VCALENDAR"/></C:filter></C:calendar-query>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 207 {
		t.Fatalf("default-depth calendar-query status=%d body=%s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), "event.ics") || strings.Contains(string(body), "task.ics") {
		t.Fatalf("Depth: 0 calendar-query unexpectedly included collection members: %s", body)
	}
}

func TestCalendarQueryRejectsUnsupportedTimeRange(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarQueryResources(t, ts.URL, client)

	report := `<C:calendar-query xmlns:C="urn:ietf:params:xml:ns:caldav"><C:filter><C:comp-filter name="VCALENDAR"><C:comp-filter name="VEVENT"><C:time-range start="20260901T000000Z" end="20261001T000000Z"/></C:comp-filter></C:comp-filter></C:filter></C:calendar-query>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, map[string]string{"Depth": "1"})
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("unsupported time-range status=%d body=%s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), "event.ics") || strings.Contains(string(body), "task.ics") {
		t.Fatalf("unsupported calendar filter leaked broadened results: %s", body)
	}
}

func TestCalendarQueryMissingCollectionStillReturnsNotFound(t *testing.T) {
	ts := newTestServer(t)
	report := `<C:calendar-query xmlns:C="urn:ietf:params:xml:ns:caldav"><C:filter><C:comp-filter name="VCALENDAR"/></C:filter></C:calendar-query>`
	resp := request(t, ts.Client(), "REPORT", ts.URL+"/dav/calendars/alice/missing/", report, nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("query against missing collection status=%d", resp.StatusCode)
	}
}

func TestAddressBookQueryRequiresDepth(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedAddressBookQueryResources(t, ts.URL, client)

	report := `<C:addressbook-query xmlns:C="urn:ietf:params:xml:ns:carddav"><C:filter><C:prop-filter name="FN"/></C:filter></C:addressbook-query>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/addressbooks/alice/contacts/", report, nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("addressbook-query without Depth status=%d", resp.StatusCode)
	}
}

func TestAddressBookQueryPropertyExistenceAndGroupedProperty(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedAddressBookQueryResources(t, ts.URL, client)

	report := `<C:addressbook-query xmlns:C="urn:ietf:params:xml:ns:carddav"><C:filter><C:prop-filter name="EMAIL"/></C:filter></C:addressbook-query>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/addressbooks/alice/contacts/", report, map[string]string{"Depth": "1"})
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 {
		t.Fatalf("addressbook-query status=%d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(text, "email.vcf") {
		t.Fatalf("EMAIL filter did not match grouped item1.EMAIL property: %s", body)
	}
	if strings.Contains(text, "phone.vcf") {
		t.Fatalf("EMAIL filter included contact without EMAIL property: %s", body)
	}
	if strings.Contains(text, "person@example.test") || strings.Contains(text, "+15555550100") {
		t.Fatalf("query without DAV property selection returned unrequested vCard data: %s", body)
	}
}

func TestAddressBookQueryAllOfPropertyExistence(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedAddressBookQueryResources(t, ts.URL, client)

	report := `<C:addressbook-query xmlns:C="urn:ietf:params:xml:ns:carddav"><C:filter test="allof"><C:prop-filter name="FN"/><C:prop-filter name="EMAIL"/></C:filter></C:addressbook-query>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/addressbooks/alice/contacts/", report, map[string]string{"Depth": "1"})
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)
	if resp.StatusCode != 207 || !strings.Contains(text, "email.vcf") {
		t.Fatalf("allof addressbook-query status=%d body=%s", resp.StatusCode, body)
	}
	if strings.Contains(text, "phone.vcf") {
		t.Fatalf("allof query included non-matching resource: %s", body)
	}
}

func TestAddressBookQueryRejectsUnsupportedTextMatch(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedAddressBookQueryResources(t, ts.URL, client)

	report := `<C:addressbook-query xmlns:C="urn:ietf:params:xml:ns:carddav"><C:filter><C:prop-filter name="FN"><C:text-match>Person</C:text-match></C:prop-filter></C:filter></C:addressbook-query>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/addressbooks/alice/contacts/", report, map[string]string{"Depth": "1"})
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("unsupported text-match status=%d body=%s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), "email.vcf") || strings.Contains(string(body), "phone.vcf") {
		t.Fatalf("unsupported CardDAV filter leaked broadened results: %s", body)
	}
}

func TestQueryRejectsInfiniteDepth(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	seedCalendarQueryResources(t, ts.URL, client)

	report := `<C:calendar-query xmlns:C="urn:ietf:params:xml:ns:caldav"><C:filter><C:comp-filter name="VCALENDAR"/></C:filter></C:calendar-query>`
	resp := request(t, client, "REPORT", ts.URL+"/dav/calendars/alice/personal/", report, map[string]string{"Depth": "infinity"})
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("infinite query Depth status=%d", resp.StatusCode)
	}
}
