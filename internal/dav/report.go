package dav

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/GoreeCloud/goreecloud-dav/internal/storage"
)

var (
	errUnsupportedReportFilter = errors.New("unsupported report filter")
	errUnsupportedReportDepth  = errors.New("unsupported report depth")
)

type reportRequest struct {
	Name              string
	Hrefs             []string
	CalendarFilter    *calendarComponentFilter
	AddressBookFilter *addressBookFilter
}

type calendarComponentFilter struct {
	Name     string
	Children []calendarComponentFilter
}

type addressBookFilter struct {
	Test       string
	Properties []string
}

type reportXMLNode struct {
	Name     xml.Name
	Attrs    []xml.Attr
	Children []*reportXMLNode
	Text     string
}

func parseReportRequest(data []byte) (reportRequest, error) {
	root, err := parseReportXML(data)
	if err != nil {
		return reportRequest{}, err
	}

	req := reportRequest{Name: root.Name.Local}
	var expectedNamespace string
	switch root.Name.Local {
	case "calendar-query", "calendar-multiget":
		expectedNamespace = nsCalDAV
	case "addressbook-query", "addressbook-multiget":
		expectedNamespace = nsCardDAV
	default:
		return reportRequest{}, fmt.Errorf("unsupported report %q", root.Name.Local)
	}
	if root.Name.Space != expectedNamespace {
		return reportRequest{}, fmt.Errorf("report has unexpected namespace %q", root.Name.Space)
	}

	for _, child := range root.Children {
		switch {
		case child.Name.Space == nsDAV && child.Name.Local == "href":
			req.Hrefs = append(req.Hrefs, strings.TrimSpace(child.Text))
		case child.Name.Space == nsDAV && (child.Name.Local == "prop" || child.Name.Local == "allprop" || child.Name.Local == "propname"):
			// Property selection is accepted here and remains a separate
			// foundation concern. The current report response still returns the
			// supported baseline report properties until selective REPORT
			// property projection is implemented.
		case root.Name.Local == "calendar-query" && child.Name.Space == nsCalDAV && child.Name.Local == "filter":
			if req.CalendarFilter != nil {
				return reportRequest{}, fmt.Errorf("calendar-query contains multiple filters")
			}
			filter, err := parseCalendarFilter(child)
			if err != nil {
				return reportRequest{}, err
			}
			req.CalendarFilter = &filter
		case root.Name.Local == "addressbook-query" && child.Name.Space == nsCardDAV && child.Name.Local == "filter":
			if req.AddressBookFilter != nil {
				return reportRequest{}, fmt.Errorf("addressbook-query contains multiple filters")
			}
			filter, err := parseAddressBookFilter(child)
			if err != nil {
				return reportRequest{}, err
			}
			req.AddressBookFilter = &filter
		default:
			return reportRequest{}, fmt.Errorf("unsupported report child %s:%s", child.Name.Space, child.Name.Local)
		}
	}

	switch req.Name {
	case "calendar-query":
		if req.CalendarFilter == nil {
			return reportRequest{}, fmt.Errorf("calendar-query requires a CALDAV:filter")
		}
	case "addressbook-query":
		if req.AddressBookFilter == nil {
			return reportRequest{}, fmt.Errorf("addressbook-query requires a CARDDAV:filter")
		}
	case "calendar-multiget", "addressbook-multiget":
		if len(req.Hrefs) == 0 {
			return reportRequest{}, fmt.Errorf("multiget requires at least one DAV:href")
		}
	}

	return req, nil
}

func parseReportXML(data []byte) (*reportXMLNode, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var root *reportXMLNode
	stack := make([]*reportXMLNode, 0, 8)

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			node := &reportXMLNode{Name: t.Name, Attrs: append([]xml.Attr(nil), t.Attr...)}
			if len(stack) == 0 {
				if root != nil {
					return nil, fmt.Errorf("multiple report roots")
				}
				root = node
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			}
			stack = append(stack, node)
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].Text += string(t)
			}
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("unexpected report end element")
			}
			stack = stack[:len(stack)-1]
		}
	}
	if root == nil || len(stack) != 0 {
		return nil, fmt.Errorf("incomplete report document")
	}
	return root, nil
}

func parseCalendarFilter(node *reportXMLNode) (calendarComponentFilter, error) {
	if strings.TrimSpace(node.Text) != "" || len(node.Children) != 1 {
		return calendarComponentFilter{}, fmt.Errorf("invalid CALDAV:filter")
	}
	child := node.Children[0]
	if child.Name.Space != nsCalDAV || child.Name.Local != "comp-filter" {
		return calendarComponentFilter{}, fmt.Errorf("CALDAV:filter requires comp-filter")
	}
	filter, err := parseCalendarComponentFilter(child)
	if err != nil {
		return calendarComponentFilter{}, err
	}
	if !strings.EqualFold(filter.Name, "VCALENDAR") {
		return calendarComponentFilter{}, fmt.Errorf("top-level calendar comp-filter must target VCALENDAR")
	}
	return filter, nil
}

func parseCalendarComponentFilter(node *reportXMLNode) (calendarComponentFilter, error) {
	name := strings.TrimSpace(reportAttr(node, "", "name"))
	if name == "" {
		return calendarComponentFilter{}, fmt.Errorf("CALDAV:comp-filter requires name")
	}
	if strings.TrimSpace(node.Text) != "" {
		return calendarComponentFilter{}, fmt.Errorf("CALDAV:comp-filter contains unexpected text")
	}

	filter := calendarComponentFilter{Name: strings.ToUpper(name)}
	for _, child := range node.Children {
		if child.Name.Space != nsCalDAV || child.Name.Local != "comp-filter" {
			return calendarComponentFilter{}, fmt.Errorf("%w: only component-existence CalDAV filters are implemented", errUnsupportedReportFilter)
		}
		nested, err := parseCalendarComponentFilter(child)
		if err != nil {
			return calendarComponentFilter{}, err
		}
		filter.Children = append(filter.Children, nested)
	}
	return filter, nil
}

func parseAddressBookFilter(node *reportXMLNode) (addressBookFilter, error) {
	if strings.TrimSpace(node.Text) != "" {
		return addressBookFilter{}, fmt.Errorf("invalid CARDDAV:filter text")
	}
	test := strings.ToLower(strings.TrimSpace(reportAttr(node, "", "test")))
	if test == "" {
		test = "anyof"
	}
	if test != "anyof" && test != "allof" {
		return addressBookFilter{}, fmt.Errorf("invalid CARDDAV filter test %q", test)
	}

	filter := addressBookFilter{Test: test}
	for _, child := range node.Children {
		if child.Name.Space != nsCardDAV || child.Name.Local != "prop-filter" {
			return addressBookFilter{}, fmt.Errorf("%w: only property-existence CardDAV filters are implemented", errUnsupportedReportFilter)
		}
		if strings.TrimSpace(child.Text) != "" || len(child.Children) != 0 {
			return addressBookFilter{}, fmt.Errorf("%w: CardDAV text/parameter/negative filters are not implemented", errUnsupportedReportFilter)
		}
		name := strings.TrimSpace(reportAttr(child, "", "name"))
		if name == "" {
			return addressBookFilter{}, fmt.Errorf("CARDDAV:prop-filter requires name")
		}
		filter.Properties = append(filter.Properties, strings.ToUpper(name))
	}
	if len(filter.Properties) == 0 {
		return addressBookFilter{}, fmt.Errorf("%w: empty CardDAV filters are not implemented", errUnsupportedReportFilter)
	}
	return filter, nil
}

func reportAttr(node *reportXMLNode, namespace, local string) string {
	for _, attr := range node.Attrs {
		if attr.Name.Space == namespace && attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}

func reportQueryIncludesMembers(req reportRequest, rawDepth string) (bool, error) {
	depth := strings.TrimSpace(rawDepth)
	switch req.Name {
	case "calendar-query":
		if depth == "" {
			depth = "0"
		}
	case "addressbook-query":
		if depth == "" {
			return false, fmt.Errorf("addressbook-query requires Depth header")
		}
	default:
		return true, nil
	}

	switch depth {
	case "0":
		return false, nil
	case "1":
		return true, nil
	case "infinity":
		return false, fmt.Errorf("%w: infinite-depth query REPORT is not implemented", errUnsupportedReportDepth)
	default:
		return false, fmt.Errorf("invalid query REPORT Depth header")
	}
}

func reportQueryMatches(req reportRequest, resource storage.Resource) bool {
	switch req.Name {
	case "calendar-query":
		return req.CalendarFilter != nil && calendarComponentFilterMatches(resource.Data, *req.CalendarFilter)
	case "addressbook-query":
		return req.AddressBookFilter != nil && addressBookFilterMatches(resource.Data, *req.AddressBookFilter)
	default:
		return true
	}
}

func calendarComponentFilterMatches(data []byte, filter calendarComponentFilter) bool {
	upper := strings.ToUpper(string(data))
	if !strings.Contains(upper, "BEGIN:"+filter.Name) || !strings.Contains(upper, "END:"+filter.Name) {
		return false
	}
	for _, child := range filter.Children {
		if !calendarComponentFilterMatches(data, child) {
			return false
		}
	}
	return true
}

func addressBookFilterMatches(data []byte, filter addressBookFilter) bool {
	matches := 0
	for _, property := range filter.Properties {
		if vcardHasProperty(data, property) {
			matches++
			if filter.Test == "anyof" {
				return true
			}
		} else if filter.Test == "allof" {
			return false
		}
	}
	if filter.Test == "allof" {
		return matches == len(filter.Properties)
	}
	return false
}

func vcardHasProperty(data []byte, wanted string) bool {
	wanted = strings.ToUpper(strings.TrimSpace(wanted))
	if wanted == "" {
		return false
	}
	for _, line := range unfoldTextLines(string(data)) {
		colon := strings.IndexByte(line, ':')
		if colon <= 0 {
			continue
		}
		left := line[:colon]
		if semi := strings.IndexByte(left, ';'); semi >= 0 {
			left = left[:semi]
		}
		left = strings.ToUpper(strings.TrimSpace(left))
		if strings.Contains(wanted, ".") {
			if left == wanted {
				return true
			}
			continue
		}
		if dot := strings.LastIndexByte(left, '.'); dot >= 0 {
			left = left[dot+1:]
		}
		if left == wanted {
			return true
		}
	}
	return false
}

func unfoldTextLines(data string) []string {
	data = strings.ReplaceAll(data, "\r\n", "\n")
	data = strings.ReplaceAll(data, "\r", "\n")
	raw := strings.Split(data, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && len(out) > 0 {
			out[len(out)-1] += line[1:]
			continue
		}
		out = append(out, line)
	}
	return out
}
