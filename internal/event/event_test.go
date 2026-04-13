package event

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestEnvelopeType(t *testing.T) {
	t.Parallel()

	if got := (Envelope{}).Type(); got != "" {
		t.Fatalf("Type() = %q, want empty", got)
	}
	if got, want := (Envelope{Message: testMessage{typ: " test.event "}}).Type(), Type(" test.event "); got != want {
		t.Fatalf("Type() = %q, want %q", got, want)
	}
}

func TestNewSpecsAndDefinition(t *testing.T) {
	t.Parallel()

	core := NewCoreSpec(testMessage{typ: "core.event"}, Identity[testMessage], nil)
	if got, want := core.Definition().Type, Type("core.event"); got != want {
		t.Fatalf("core type = %q, want %q", got, want)
	}
	if got, want := core.Definition().Owner, OwnerCore; got != want {
		t.Fatalf("core owner = %q, want %q", got, want)
	}

	system := NewSystemSpec(testMessage{typ: "sys.demo.event"}, "demo", Identity[testMessage], nil)
	if got, want := system.Definition().SystemID, "demo"; got != want {
		t.Fatalf("system id = %q, want %q", got, want)
	}
}

func TestTypedSpecValidateMessage(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		spec := NewCoreSpec(testMessage{typ: "test.event"}, Identity[testMessage], func(message testMessage) error {
			if message.name != "ok" {
				return errors.New("bad name")
			}
			return nil
		})
		if err := spec.ValidateMessage(testMessage{typ: "test.event", name: "ok"}); err != nil {
			t.Fatalf("ValidateMessage() error = %v", err)
		}
	})

	t.Run("mistyped", func(t *testing.T) {
		spec := NewCoreSpec(testMessage{typ: "test.event"}, Identity[testMessage], nil)
		if err := spec.ValidateMessage(otherMessage{}); err == nil {
			t.Fatal("ValidateMessage() error = nil, want failure")
		}
	})

	t.Run("nil validator", func(t *testing.T) {
		spec := NewCoreSpec(testMessage{typ: "test.event"}, Identity[testMessage], nil)
		if err := spec.ValidateMessage(testMessage{typ: "test.event"}); err != nil {
			t.Fatalf("ValidateMessage() error = %v", err)
		}
	})
}

func TestTypedSpecNormalizeMessage(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		spec := NewCoreSpec(testMessage{typ: "test.event"}, func(message testMessage) testMessage {
			message.name = strings.TrimSpace(message.name)
			return message
		}, nil)
		normalized, err := spec.NormalizeMessage(testMessage{typ: "test.event", name: " ok "})
		if err != nil {
			t.Fatalf("NormalizeMessage() error = %v", err)
		}
		message, ok := normalized.(testMessage)
		if !ok {
			t.Fatalf("normalized type = %T, want testMessage", normalized)
		}
		if got, want := message.name, "ok"; got != want {
			t.Fatalf("name = %q, want %q", got, want)
		}
	})

	t.Run("mistyped", func(t *testing.T) {
		spec := NewCoreSpec(testMessage{typ: "test.event"}, Identity[testMessage], nil)
		if _, err := spec.NormalizeMessage(otherMessage{}); err == nil {
			t.Fatal("NormalizeMessage() error = nil, want failure")
		}
	})

	t.Run("identity", func(t *testing.T) {
		if got, want := Identity(testMessage{typ: "test.event", name: "ok"}).name, "ok"; got != want {
			t.Fatalf("Identity() name = %q, want %q", got, want)
		}
	})
}

func TestNewSpecRequiresNormalizer(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("NewCoreSpec() panic = nil, want failure")
		}
	}()

	var normalize func(testMessage) testMessage
	_ = NewCoreSpec(testMessage{typ: "test.event"}, normalize, nil)
}

func TestNewEnvelopeAndMessageAs(t *testing.T) {
	t.Parallel()

	spec := NewCoreSpec(testMessage{typ: "test.event"}, func(message testMessage) testMessage {
		message.name = strings.TrimSpace(message.name)
		return message
	}, func(message testMessage) error {
		if message.name != "ok" {
			return errors.New("name must be normalized")
		}
		return nil
	})
	if _, err := NewEnvelope(spec, "", testMessage{typ: "test.event", name: "ok"}); err == nil {
		t.Fatal("NewEnvelope() should reject blank campaign id")
	}
	badSpec := TypedSpec[testMessage]{
		definition: Definition{
			Type:        "bad.event",
			Owner:       Owner("wat"),
			MessageType: reflect.TypeFor[testMessage](),
		},
	}
	if _, err := NewEnvelope(badSpec, "camp-1", testMessage{typ: "test.event", name: "ok"}); err == nil {
		t.Fatal("NewEnvelope() should reject invalid specs")
	}
	if _, err := NewEnvelope(spec, "camp-1", testMessage{typ: "test.event", name: ""}); err == nil {
		t.Fatal("NewEnvelope() should reject invalid message")
	}
	if _, err := NewEnvelope(spec, "camp-1", testMessage{typ: "other.event", name: "ok"}); err == nil {
		t.Fatal("NewEnvelope() should reject mismatched event types")
	}

	nilSpec := NewCoreSpec[Message](testMessage{typ: "test.event"}, Identity[Message], nil)
	var nilMessage Message
	if _, err := NewEnvelope(nilSpec, "camp-1", nilMessage); err == nil {
		t.Fatal("NewEnvelope() should reject nil messages")
	}

	envelope, err := NewEnvelope(spec, "camp-1", testMessage{typ: "test.event", name: " ok "})
	if err != nil {
		t.Fatalf("NewEnvelope() error = %v", err)
	}
	if got, want := envelope.CampaignID, "camp-1"; got != want {
		t.Fatalf("campaign id = %q, want %q", got, want)
	}

	if _, err := MessageAs[testMessage](Envelope{Message: otherMessage{}}); err == nil {
		t.Fatal("MessageAs() error = nil, want failure")
	}
	message, err := MessageAs[testMessage](envelope)
	if err != nil {
		t.Fatalf("MessageAs() error = %v", err)
	}
	if got, want := message.name, "ok"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
}

func TestNewCatalog(t *testing.T) {
	t.Parallel()

	validType := Type("test.event")
	validTypeOf := reflect.TypeFor[testMessage]()

	tests := []struct {
		name string
		spec Spec
	}{
		{name: "nil spec", spec: nil},
		{name: "missing type", spec: stubSpec{def: Definition{Owner: OwnerCore, MessageType: validTypeOf}}},
		{name: "missing message type", spec: stubSpec{def: Definition{Type: validType, Owner: OwnerCore}}},
		{name: "invalid owner", spec: stubSpec{def: Definition{Type: validType, Owner: Owner("wat"), MessageType: validTypeOf}}},
		{name: "core sys namespace", spec: stubSpec{def: Definition{Type: "sys.demo.event", Owner: OwnerCore, MessageType: validTypeOf}}},
		{name: "system missing id", spec: stubSpec{def: Definition{Type: "sys.demo.event", Owner: OwnerSystem, MessageType: validTypeOf}}},
		{name: "system namespace mismatch", spec: stubSpec{def: Definition{Type: "sys.other.event", Owner: OwnerSystem, SystemID: "demo", MessageType: validTypeOf}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewCatalog(test.spec); err == nil {
				t.Fatal("NewCatalog() error = nil, want failure")
			}
		})
	}

	validSpec := NewCoreSpec(testMessage{typ: "test.event"}, Identity[testMessage], nil)
	if _, err := NewCatalog(validSpec, validSpec); err == nil {
		t.Fatal("NewCatalog() should reject duplicate types")
	}

	catalog, err := NewCatalog(validSpec)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	if _, ok := catalog.specs[validSpec.Definition().Type]; !ok {
		t.Fatal("catalog should contain registered spec")
	}
}

func TestCatalogValidate(t *testing.T) {
	t.Parallel()

	t.Run("nil catalog", func(t *testing.T) {
		var catalog *Catalog
		if _, _, err := catalog.Validate(Envelope{}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})

	spec := NewCoreSpec(testMessage{typ: "test.event"}, func(message testMessage) testMessage {
		message.name = strings.TrimSpace(message.name)
		return message
	}, func(message testMessage) error {
		if message.name != "ok" {
			return errors.New("name must be normalized")
		}
		return nil
	})
	catalog, err := NewCatalog(spec)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	t.Run("missing campaign id", func(t *testing.T) {
		if _, _, err := catalog.Validate(Envelope{Message: testMessage{typ: "test.event", name: "ok"}}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})

	t.Run("nil message", func(t *testing.T) {
		if _, _, err := catalog.Validate(Envelope{CampaignID: "camp-1"}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		if _, _, err := catalog.Validate(Envelope{CampaignID: "camp-1", Message: testMessage{typ: "missing.event", name: "ok"}}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})

	t.Run("message validation failure", func(t *testing.T) {
		if _, _, err := catalog.Validate(Envelope{CampaignID: "camp-1", Message: testMessage{typ: "test.event"}}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})

	t.Run("success", func(t *testing.T) {
		envelope, matched, err := catalog.Validate(Envelope{
			CampaignID: "camp-1",
			Message:    testMessage{typ: "test.event", name: " ok "},
		})
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if got, want := envelope.CampaignID, "camp-1"; got != want {
			t.Fatalf("campaign id = %q, want %q", got, want)
		}
		if got, want := matched.Definition().Type, Type("test.event"); got != want {
			t.Fatalf("spec type = %q, want %q", got, want)
		}
		message, err := MessageAs[testMessage](envelope)
		if err != nil {
			t.Fatalf("MessageAs() error = %v", err)
		}
		if got, want := message.name, "ok"; got != want {
			t.Fatalf("message name = %q, want %q", got, want)
		}
	})

	t.Run("message normalization failure", func(t *testing.T) {
		manual := &Catalog{specs: map[Type]Spec{
			"test.event": stubSpec{
				def:          Definition{Type: "test.event", Owner: OwnerCore, MessageType: reflect.TypeFor[testMessage]()},
				normalizeErr: errors.New("boom"),
			},
		}}
		if _, _, err := manual.Validate(Envelope{CampaignID: "camp-1", Message: testMessage{typ: "test.event", name: "ok"}}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})
}

func TestCatalogSpecFor(t *testing.T) {
	t.Parallel()

	var nilCatalog *Catalog
	if spec, ok := nilCatalog.SpecFor("test.event"); ok || spec != nil {
		t.Fatalf("nil catalog SpecFor() = (%v,%t), want (nil,false)", spec, ok)
	}

	spec := NewCoreSpec(testMessage{typ: "test.event"}, Identity[testMessage], nil)
	catalog, err := NewCatalog(spec)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	if _, ok := catalog.SpecFor("missing.event"); ok {
		t.Fatal("SpecFor(missing) = found, want missing")
	}
	matched, ok := catalog.SpecFor("test.event")
	if !ok {
		t.Fatal("SpecFor(test.event) = missing, want match")
	}
	if got, want := matched.Definition().Type, Type("test.event"); got != want {
		t.Fatalf("SpecFor() type = %q, want %q", got, want)
	}

	if _, ok := catalog.SpecFor(" test.event "); ok {
		t.Fatal("SpecFor(padded) = found, want missing")
	}
}

type testMessage struct {
	typ  Type
	name string
}

func (m testMessage) EventType() Type { return m.typ }

type otherMessage struct{}

func (otherMessage) EventType() Type { return "other.event" }

type stubSpec struct {
	def          Definition
	normalizeErr error
	err          error
}

func (s stubSpec) Definition() Definition { return s.def }
func (s stubSpec) NormalizeMessage(message Message) (Message, error) {
	if s.normalizeErr != nil {
		return nil, s.normalizeErr
	}
	return message, nil
}
func (s stubSpec) ValidateMessage(Message) error { return s.err }
