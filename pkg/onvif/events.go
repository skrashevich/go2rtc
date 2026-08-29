package onvif

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// WS-Addressing action URIs for the subscription manager operations. Some
// cameras (Reolink) reject a request to the subscription manager that does
// not carry wsa:To and wsa:Action, answering 400 with an empty fault.
const (
	actionPullMessages = "http://www.onvif.org/ver10/events/wsdl/PullPointSubscription/PullMessagesRequest"
	actionRenew        = "http://docs.oasis-open.org/wsn/bw-2/SubscriptionManager/RenewRequest"
	actionUnsubscribe  = "http://docs.oasis-open.org/wsn/bw-2/SubscriptionManager/UnsubscribeRequest"
)

// EventSubscription holds state for an ONVIF PullPoint event subscription.
type EventSubscription struct {
	client *Client
	// address is the PullPoint subscription manager URL (from CreatePullPointSubscription response).
	address string
	// rawAddress is the address exactly as the camera reported it, used for
	// wsa:To. The camera matches this against what it issued, so the
	// reachability rewrite applied to `address` must not leak into it.
	rawAddress string
	// refParams is the inner XML of the subscription's ReferenceParameters,
	// echoed back as SOAP headers on every subsequent request. Axis puts the
	// subscription's identity here (a SubscriptionId) and returns the same
	// URL for every subscription, so dropping this makes the camera unable
	// to tell which subscription is being polled, answering 400.
	refParams string
}

// headers builds the SOAP headers a subscription-manager request needs.
func (s *EventSubscription) headers(action string) string {
	return s.refParams +
		`<wsa:To s:mustUnderstand="1">` + s.rawAddress + `</wsa:To>` +
		`<wsa:Action s:mustUnderstand="1">` + action + `</wsa:Action>`
}

// CreatePullPointSubscription creates an ONVIF PullPoint subscription on the camera's event service.
// The timeout specifies how long the subscription stays alive before needing renewal.
func (c *Client) CreatePullPointSubscription(timeout time.Duration) (*EventSubscription, error) {
	if c.eventURL == "" {
		return nil, errors.New("onvif: event service not available")
	}

	secs := int(timeout.Seconds())
	if secs < 10 {
		secs = 10
	}

	log.Debug().Str("event_url", c.eventURL).Int("timeout_secs", secs).
		Msg("[onvif] creating pull point subscription")

	body := fmt.Sprintf(`<tev:CreatePullPointSubscription>`+
		`<tev:InitialTerminationTime>PT%dS</tev:InitialTerminationTime>`+
		`</tev:CreatePullPointSubscription>`, secs)

	b, err := c.EventRequest(c.eventURL, body)
	if err != nil {
		return nil, fmt.Errorf("onvif: create pull point: %w", err)
	}

	log.Trace().Str("response", string(b)).Msg("[onvif] create pull point response")

	// Extract subscription reference address from response.
	// Response contains: <wsnt:SubscriptionReference><wsa:Address>http://...</wsa:Address>
	addr := FindTagValue(b, "Address")
	if addr == "" {
		return nil, errors.New("onvif: no subscription address in response")
	}

	log.Debug().Str("raw_address", addr).Msg("[onvif] subscription address from camera")

	// Some cameras return relative paths or localhost — fix using camera's host.
	resolved := c.resolveEventAddress(addr)

	log.Debug().Str("resolved_address", resolved).Msg("[onvif] subscription address resolved")

	refParams := findReferenceParameters(b)
	if refParams != "" {
		log.Debug().Str("ref_params", refParams).Msg("[onvif] subscription reference parameters")
	}

	return &EventSubscription{
		client:     c,
		address:    resolved,
		rawAddress: addr,
		refParams:  refParams,
	}, nil
}

var reRefParams = regexp.MustCompile(`(?s)<([a-zA-Z0-9]+:)?ReferenceParameters[^>]*>(.*?)</([a-zA-Z0-9]+:)?ReferenceParameters>`)

// findReferenceParameters returns the inner XML of the subscription's
// ReferenceParameters, if the camera supplied any. Per WS-Addressing these
// must be copied verbatim into the headers of messages sent to the endpoint.
func findReferenceParameters(b []byte) string {
	if m := reRefParams.FindSubmatch(b); m != nil {
		return strings.TrimSpace(string(m[2]))
	}
	return ""
}

// PullMessages polls the camera for events. This is a long-poll: it blocks
// up to the specified timeout waiting for events. Returns raw XML response.
func (s *EventSubscription) PullMessages(timeout time.Duration, limit int) ([]byte, error) {
	secs := int(timeout.Seconds())
	if secs < 1 {
		secs = 1
	}
	if limit < 1 {
		limit = 1
	}

	body := fmt.Sprintf(`<tev:PullMessages>`+
		`<tev:Timeout>PT%dS</tev:Timeout>`+
		`<tev:MessageLimit>%d</tev:MessageLimit>`+
		`</tev:PullMessages>`, secs, limit)

	return s.client.eventRequest(s.address, s.headers(actionPullMessages), body)
}

// Renew extends the subscription lifetime by the specified duration.
func (s *EventSubscription) Renew(timeout time.Duration) error {
	secs := int(timeout.Seconds())

	log.Trace().Str("address", s.address).Int("timeout_secs", secs).
		Msg("[onvif] renewing subscription")

	body := fmt.Sprintf(`<wsnt:Renew>`+
		`<wsnt:TerminationTime>PT%dS</wsnt:TerminationTime>`+
		`</wsnt:Renew>`, secs)

	_, err := s.client.eventRequest(s.address, s.headers(actionRenew), body)
	return err
}

// Unsubscribe terminates the subscription on the camera (best-effort).
func (s *EventSubscription) Unsubscribe() error {
	log.Trace().Str("address", s.address).Msg("[onvif] unsubscribing")
	_, err := s.client.eventRequest(s.address, s.headers(actionUnsubscribe), `<wsnt:Unsubscribe/>`)
	return err
}

// EventRequest sends a SOAP request with event-specific namespaces.
func (c *Client) EventRequest(reqURL, body string) ([]byte, error) {
	return c.eventRequest(reqURL, "", body)
}

// eventRequest sends a SOAP request, optionally with extra SOAP headers
// alongside the security header (WS-Addressing, reference parameters).
func (c *Client) eventRequest(reqURL, extraHeaders, body string) ([]byte, error) {
	if reqURL == "" {
		return nil, errors.New("onvif: unsupported service")
	}

	e := NewEventEnvelopeWithHeaders(c.url.User, extraHeaders)
	e.Append(body)

	log.Trace().Str("url", reqURL).Msg("[onvif] event request sending")

	// Use a longer timeout for PullMessages (long-poll).
	client := &http.Client{Timeout: 90 * time.Second}
	res, err := client.Post(reqURL, `application/soap+xml;charset=utf-8`, bytes.NewReader(e.Bytes()))
	if err != nil {
		log.Trace().Err(err).Str("url", reqURL).Msg("[onvif] event request failed")
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Debug().Str("url", reqURL).Int("status", res.StatusCode).
			Msg("[onvif] event request non-200 response")
		return nil, errors.New("onvif: event request failed " + res.Status)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	log.Trace().Str("url", reqURL).Int("bytes", len(b)).
		Msg("[onvif] event request response received")

	return b, nil
}

// resolveEventAddress fixes subscription addresses returned by the camera.
// The camera may return its internal IP, localhost, or a relative path.
// We always use the host from the original client URL since we know it's reachable.
func (c *Client) resolveEventAddress(addr string) string {
	u, err := url.Parse(addr)
	if err != nil {
		return addr
	}

	// Always use the host we connected to (handles Docker, NAT, port mapping, etc.).
	u.Host = c.url.Host

	if u.Scheme == "" {
		u.Scheme = "http"
	}

	return u.String()
}

// DefaultMotionItems are the SimpleItem names that carry a motion state.
// "active" covers the Axis application topics (Object Analytics and other
// ACAP scenarios), which report their state under that name rather than the
// IsMotion/State the plain ONVIF motion topics use.
var DefaultMotionItems = []string{"IsMotion", "State", "active"}

// ParseMotionEvents extracts motion state from a PullMessages response using
// the default motion topics and item names.
// Returns (motionDetected, found). If no matching notification is present, found=false.
//
// Recognizes common ONVIF motion event topics:
//   - tns1:RuleEngine/CellMotionDetector/Motion (IsMotion property)
//   - tns1:VideoSource/MotionAlarm (State property)
//   - tns1:RuleEngine/MotionRegionDetector/Motion
func ParseMotionEvents(b []byte) (motion bool, found bool) {
	return ParseEvents(b, "", nil)
}

// ParseEvents extracts a boolean state from a PullMessages response.
//
// topicMatch selects which notifications count: a case-insensitive substring
// of the event topic. Empty means the built-in motion topics, which is what
// ParseMotionEvents uses. Naming a topic is how a camera's own analytics get
// used instead, e.g. "ObjectAnalytics/Device1Scenario1" on an Axis.
//
// items are the SimpleItem names that carry the state; nil means
// DefaultMotionItems.
func ParseEvents(b []byte, topicMatch string, items []string) (motion bool, found bool) {
	s := string(b)

	if len(items) == 0 {
		items = DefaultMotionItems
	}

	reTopic := regexp.MustCompile(`(?s)<[^>]*Topic[^>]*>([^<]*)</`)

	topics := reTopic.FindAllStringSubmatch(s, -1)
	if len(topics) == 0 {
		log.Trace().Msg("[onvif] parse: no topics found in response")
		return false, false
	}

	log.Trace().Int("topic_count", len(topics)).Msg("[onvif] parse: topics found")
	for i, t := range topics {
		if len(t) >= 2 {
			log.Trace().Int("idx", i).Str("topic", t[1]).Msg("[onvif] parse: topic")
		}
	}

	// Split response into individual NotificationMessage blocks.
	messages := splitNotificationMessages(s)

	log.Trace().Int("message_count", len(messages)).Msg("[onvif] parse: notification messages")

	for _, msg := range messages {
		topicNode := reTopic.FindStringSubmatch(msg)
		if len(topicNode) < 2 {
			log.Trace().Msg("[onvif] parse: message has no topic, skipping")
			continue
		}
		topic := topicNode[1]

		if !matchTopic(topic, topicMatch) {
			log.Trace().Str("topic", topic).Msg("[onvif] parse: topic not selected, skipping")
			continue
		}

		log.Trace().Str("topic", topic).Msg("[onvif] parse: topic selected")

		name, val, ok := findSimpleItem(msg, items)
		if !ok {
			log.Trace().Str("topic", topic).Strs("wanted", items).
				Msg("[onvif] parse: no state item in message")
			continue
		}

		val = strings.ToLower(strings.TrimSpace(val))
		motion = val == "true" || val == "1"
		found = true

		log.Trace().Str("topic", topic).Str("name", name).
			Str("value", val).Bool("motion", motion).
			Msg("[onvif] parse: state extracted")
		// Use the last matching event if several are present.
	}

	return motion, found
}

var (
	// A SimpleItem element, capturing its attribute list. Attributes are read
	// separately rather than matched in order, because Name and Value appear
	// in either order depending on the camera (Tapo sends Value first) and a
	// single ordered pattern silently drops half of them.
	reSimpleItem = regexp.MustCompile(`<[^>]*\bSimpleItem\b([^>]*)>`)
	reAttribute  = regexp.MustCompile(`(\w+)\s*=\s*"([^"]*)"`)
)

// findSimpleItem returns the value of the first SimpleItem whose Name is one
// of want, regardless of the order its attributes appear in.
func findSimpleItem(msg string, want []string) (name, value string, ok bool) {
	for _, item := range reSimpleItem.FindAllStringSubmatch(msg, -1) {
		var n, v string
		var haveValue bool
		for _, attr := range reAttribute.FindAllStringSubmatch(item[1], -1) {
			switch attr[1] {
			case "Name":
				n = attr[2]
			case "Value":
				v, haveValue = attr[2], true
			}
		}
		if n == "" || !haveValue {
			continue
		}
		for _, w := range want {
			if strings.EqualFold(n, w) {
				return n, v, true
			}
		}
	}
	return "", "", false
}

// matchTopic reports whether a notification topic is one we care about.
// An empty match falls back to the built-in motion topics.
func matchTopic(topic, match string) bool {
	topic = strings.ToLower(topic)
	if match != "" {
		return strings.Contains(topic, strings.ToLower(match))
	}
	return isMotionTopic(topic)
}

// isMotionTopic checks if a topic string relates to motion detection.
func isMotionTopic(topic string) bool {
	topic = strings.ToLower(topic)
	return strings.Contains(topic, "motiondetector") ||
		strings.Contains(topic, "motionalarm") ||
		strings.Contains(topic, "motionregiondetector") ||
		strings.Contains(topic, "cellmotiondetector")
}

// splitNotificationMessages splits the XML response into individual notification message blocks.
func splitNotificationMessages(s string) []string {
	re := regexp.MustCompile(`(?s)<[^>]*NotificationMessage[^>]*>.*?</[^>]*NotificationMessage>`)
	return re.FindAllString(s, -1)
}
