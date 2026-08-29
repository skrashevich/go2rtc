package onvif

import (
	"fmt"
	"testing"
)

// Payloads captured from real cameras. The attribute order in each is the
// order that camera actually sends, which is the point of these tests.
const (
	// Tapo C200. Note Value comes before Name: an ordered pattern that
	// expects Name first silently drops every one of these.
	tapoMotion = `<wsnt:NotificationMessage><wsnt:Topic Dialect="http://www.onvif.org/ver10/tev/topicExpression/ConcreteSet">tns1:RuleEngine/CellMotionDetector/Motion</wsnt:Topic><wsnt:Message><tt:Message PropertyOperation="Changed" UtcTime="1970-01-01T00:00:00Z"><tt:Source><tt:SimpleItem Value="vsconf" Name="VideoSourceConfigurationToken"></tt:SimpleItem><tt:SimpleItem Value="VideoAnalyticsToken" Name="VideoAnalyticsConfigurationToken"></tt:SimpleItem><tt:SimpleItem Value="MyMotionDetectorRule" Name="Rule"></tt:SimpleItem></tt:Source><tt:Data><tt:SimpleItem Value="%s" Name="IsMotion"></tt:SimpleItem></tt:Data></tt:Message></wsnt:Message></wsnt:NotificationMessage>`

	// Axis, plain ONVIF motion alarm. Name before Value.
	axisMotionAlarm = `<wsnt:NotificationMessage><wsnt:Topic Dialect="http://docs.oasis-open.org/wsn/t-1/TopicExpression/Simple">tns1:VideoSource/MotionAlarm</wsnt:Topic><wsnt:Message><tt:Message UtcTime="2026-08-29T18:25:09Z" PropertyOperation="Changed"><tt:Source><tt:SimpleItem Name="Source" Value="0"/></tt:Source><tt:Data><tt:SimpleItem Name="State" Value="1"/></tt:Data></tt:Message></wsnt:Message></wsnt:NotificationMessage>`

	// Axis Object Analytics. Its state item is "active", and the topic is
	// under the vendor namespace rather than any of the motion topics.
	axisAOA = `<wsnt:NotificationMessage><wsnt:Topic Dialect="http://docs.oasis-open.org/wsn/t-1/TopicExpression/Simple">tnsaxis:CameraApplicationPlatform/ObjectAnalytics/Device1Scenario1</wsnt:Topic><wsnt:Message><tt:Message UtcTime="2026-08-29T18:25:08Z" PropertyOperation="Changed"><tt:Data><tt:SimpleItem Name="active" Value="1"/><tt:SimpleItem Name="classTypes" Value="human"/><tt:SimpleItem Name="scenarioType" Value="ObjectInArea"/><tt:SimpleItem Name="objectId" Value="68600"/></tt:Data></tt:Message></wsnt:Message></wsnt:NotificationMessage>`
)

func TestParseMotionEvents_AttributeOrder(t *testing.T) {
	// Value-before-Name is equally valid XML and must parse identically.
	for _, tt := range []struct {
		name string
		xml  string
		want bool
	}{
		{"tapo motion on", fmt.Sprintf(tapoMotion, "true"), true},
		{"tapo motion off", fmt.Sprintf(tapoMotion, "false"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			motion, found := ParseMotionEvents([]byte(tt.xml))
			if !found {
				t.Fatal("event not found; the parser must not depend on attribute order")
			}
			if motion != tt.want {
				t.Fatalf("motion = %v, want %v", motion, tt.want)
			}
		})
	}
}

func TestParseMotionEvents_MotionAlarm(t *testing.T) {
	motion, found := ParseMotionEvents([]byte(axisMotionAlarm))
	if !found || !motion {
		t.Fatalf("found=%v motion=%v, want both true", found, motion)
	}
}

func TestParseEvents_AxisObjectAnalytics(t *testing.T) {
	// Not motion, so the default topics must ignore it.
	if _, found := ParseMotionEvents([]byte(axisAOA)); found {
		t.Fatal("AOA should not match the default motion topics")
	}

	// Selecting the scenario by topic picks it up; "active" is a default item.
	motion, found := ParseEvents([]byte(axisAOA), "ObjectAnalytics/Device1Scenario1", nil)
	if !found || !motion {
		t.Fatalf("found=%v motion=%v, want both true", found, motion)
	}
}

func TestParseEvents_TopicSelectsOneOfSeveral(t *testing.T) {
	// A response carrying both: the selected topic must win, not whichever
	// happens to come last.
	both := axisMotionAlarm + axisAOA

	motion, found := ParseEvents([]byte(both), "ObjectAnalytics/Device1Scenario1", nil)
	if !found || !motion {
		t.Fatalf("AOA: found=%v motion=%v, want both true", found, motion)
	}

	motion, found = ParseEvents([]byte(both), "MotionAlarm", nil)
	if !found || !motion {
		t.Fatalf("MotionAlarm: found=%v motion=%v, want both true", found, motion)
	}
}

func TestParseEvents_ExplicitItemName(t *testing.T) {
	if _, found := ParseEvents([]byte(axisAOA), "ObjectAnalytics", []string{"IsMotion"}); found {
		t.Fatal("should not match: AOA has no IsMotion item")
	}
	if _, found := ParseEvents([]byte(axisAOA), "ObjectAnalytics", []string{"active"}); !found {
		t.Fatal("should match on the named item")
	}
}
