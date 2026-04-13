package event

import (
	"reflect"
	"testing"
)

func TestCanonicalValidationBranches(t *testing.T) {
	t.Parallel()

	spec := NewCoreSpec(testMessage{typ: "test.event"}, Identity[testMessage], nil)
	if _, err := NewEnvelope(spec, " camp-1 ", testMessage{typ: "test.event", name: "ok"}); err == nil {
		t.Fatal("NewEnvelope(padded campaign id) error = nil, want failure")
	}

	catalog, err := NewCatalog(spec)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	if _, _, err := catalog.Validate(Envelope{CampaignID: " camp-1 ", Message: testMessage{typ: "test.event", name: "ok"}}); err == nil {
		t.Fatal("Validate(padded campaign id) error = nil, want failure")
	}

	messageType := reflect.TypeFor[testMessage]()
	if err := validateDefinition(Definition{
		Type:        " test.event ",
		Owner:       OwnerCore,
		MessageType: messageType,
	}); err == nil {
		t.Fatal("validateDefinition(padded type) error = nil, want failure")
	}
	if err := validateDefinition(Definition{
		Type:        "sys.demo.event",
		Owner:       OwnerSystem,
		SystemID:    " demo ",
		MessageType: messageType,
	}); err == nil {
		t.Fatal("validateDefinition(padded system id) error = nil, want failure")
	}
}
