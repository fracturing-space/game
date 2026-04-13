package command

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
	if got, want := (Envelope{Message: testMessage{typ: " test.command "}}).Type(), Type(" test.command "); got != want {
		t.Fatalf("Type() = %q, want %q", got, want)
	}
}

func TestNewSpecsAndDefinition(t *testing.T) {
	t.Parallel()

	core := NewCoreSpec(CoreSpecArgs[testMessage]{Message: testMessage{typ: "core.command"}, Scope: ScopeCampaign})
	if got, want := core.Definition().Type, Type("core.command"); got != want {
		t.Fatalf("core type = %q, want %q", got, want)
	}
	if got, want := core.Definition().Owner, OwnerCore; got != want {
		t.Fatalf("core owner = %q, want %q", got, want)
	}

	system := NewSystemSpec(SystemSpecArgs[testMessage]{Message: testMessage{typ: "sys.demo.command"}, SystemID: "demo", Scope: ScopeNewCampaign})
	if got, want := system.Definition().SystemID, "demo"; got != want {
		t.Fatalf("system id = %q, want %q", got, want)
	}
	if got, want := system.Definition().Scope, ScopeNewCampaign; got != want {
		t.Fatalf("system scope = %q, want %q", got, want)
	}
}

func TestTypedSpecValidateMessage(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		spec := NewCoreSpec(CoreSpecArgs[testMessage]{Message: testMessage{typ: "test.command"}, Scope: ScopeCampaign, Validate: func(message testMessage) error {
			if message.name != "ok" {
				return errors.New("bad name")
			}
			return nil
		}})
		if err := spec.ValidateMessage(testMessage{typ: "test.command", name: "ok"}); err != nil {
			t.Fatalf("ValidateMessage() error = %v", err)
		}
	})

	t.Run("mistyped", func(t *testing.T) {
		spec := NewCoreSpec(CoreSpecArgs[testMessage]{Message: testMessage{typ: "test.command"}, Scope: ScopeCampaign})
		if err := spec.ValidateMessage(otherMessage{}); err == nil {
			t.Fatal("ValidateMessage() error = nil, want failure")
		}
	})

	t.Run("nil validator", func(t *testing.T) {
		spec := NewCoreSpec(CoreSpecArgs[testMessage]{Message: testMessage{typ: "test.command"}, Scope: ScopeCampaign})
		if err := spec.ValidateMessage(testMessage{typ: "test.command"}); err != nil {
			t.Fatalf("ValidateMessage() error = %v", err)
		}
	})
}

func TestTypedSpecNormalizeMessage(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		spec := NewCoreSpec(CoreSpecArgs[testMessage]{Message: testMessage{typ: "test.command"}, Scope: ScopeCampaign, Normalize: func(message testMessage) testMessage {
			message.name = strings.TrimSpace(message.name)
			return message
		}})
		normalized, err := spec.NormalizeMessage(testMessage{typ: "test.command", name: " ok "})
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
		spec := NewCoreSpec(CoreSpecArgs[testMessage]{Message: testMessage{typ: "test.command"}, Scope: ScopeCampaign})
		if _, err := spec.NormalizeMessage(otherMessage{}); err == nil {
			t.Fatal("NormalizeMessage() error = nil, want failure")
		}
	})
}

func TestNewSpecDefaultsToIdentityNormalizer(t *testing.T) {
	t.Parallel()

	spec := NewCoreSpec(CoreSpecArgs[testMessage]{Message: testMessage{typ: "test.command"}, Scope: ScopeCampaign})
	normalized, err := spec.NormalizeMessage(testMessage{typ: "test.command", name: "ok"})
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
}

func TestMessageAs(t *testing.T) {
	t.Parallel()

	if _, err := MessageAs[testMessage](Envelope{Message: otherMessage{}}); err == nil {
		t.Fatal("MessageAs() error = nil, want failure")
	}
	message, err := MessageAs[testMessage](Envelope{Message: testMessage{typ: "test.command", name: "ok"}})
	if err != nil {
		t.Fatalf("MessageAs() error = %v", err)
	}
	if got, want := message.name, "ok"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
}

func TestNewCatalog(t *testing.T) {
	t.Parallel()

	validType := Type("test.command")
	validTypeOf := reflect.TypeFor[testMessage]()

	tests := []struct {
		name string
		spec Spec
	}{
		{name: "nil spec", spec: nil},
		{name: "missing type", spec: stubSpec{def: Definition{Owner: OwnerCore, Scope: ScopeCampaign, MessageType: validTypeOf}}},
		{name: "missing message type", spec: stubSpec{def: Definition{Type: validType, Owner: OwnerCore, Scope: ScopeCampaign}}},
		{name: "invalid owner", spec: stubSpec{def: Definition{Type: validType, Owner: Owner("wat"), Scope: ScopeCampaign, MessageType: validTypeOf}}},
		{name: "invalid scope", spec: stubSpec{def: Definition{Type: validType, Owner: OwnerCore, Scope: Scope("wat"), MessageType: validTypeOf}}},
		{name: "core sys namespace", spec: stubSpec{def: Definition{Type: "sys.demo.command", Owner: OwnerCore, Scope: ScopeCampaign, MessageType: validTypeOf}}},
		{name: "system missing id", spec: stubSpec{def: Definition{Type: "sys.demo.command", Owner: OwnerSystem, Scope: ScopeCampaign, MessageType: validTypeOf}}},
		{name: "system namespace mismatch", spec: stubSpec{def: Definition{Type: "sys.other.command", Owner: OwnerSystem, SystemID: "demo", Scope: ScopeCampaign, MessageType: validTypeOf}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewCatalog(test.spec); err == nil {
				t.Fatal("NewCatalog() error = nil, want failure")
			}
		})
	}

	validSpec := NewCoreSpec(CoreSpecArgs[testMessage]{Message: testMessage{typ: "test.command"}, Scope: ScopeCampaign})
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

	validSpec := NewCoreSpec(CoreSpecArgs[testMessage]{Message: testMessage{typ: "test.command"}, Scope: ScopeCampaign, Normalize: func(message testMessage) testMessage {
		message.name = strings.TrimSpace(message.name)
		return message
	}, Validate: func(message testMessage) error {
		if message.name != "ok" {
			return errors.New("name must be normalized")
		}
		return nil
	}})
	newCampaignSpec := NewCoreSpec(CoreSpecArgs[testMessage]{Message: testMessage{typ: "test.new"}, Scope: ScopeNewCampaign})
	catalog, err := NewCatalog(validSpec, newCampaignSpec)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	t.Run("nil message", func(t *testing.T) {
		if _, _, err := catalog.Validate(Envelope{CampaignID: "camp-1"}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		if _, _, err := catalog.Validate(Envelope{CampaignID: "camp-1", Message: testMessage{typ: "missing.command"}}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})

	t.Run("missing campaign id", func(t *testing.T) {
		if _, _, err := catalog.Validate(Envelope{Message: testMessage{typ: "test.command", name: "ok"}}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})

	t.Run("forbidden campaign id", func(t *testing.T) {
		if _, _, err := catalog.Validate(Envelope{CampaignID: "camp-1", Message: testMessage{typ: "test.new"}}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})

	t.Run("message validation failure", func(t *testing.T) {
		if _, _, err := catalog.Validate(Envelope{CampaignID: "camp-1", Message: testMessage{typ: "test.command"}}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})

	t.Run("invalid scope in catalog", func(t *testing.T) {
		manual := &Catalog{specs: map[Type]Spec{
			"test.command": stubSpec{def: Definition{Type: "test.command", Owner: OwnerCore, Scope: Scope("wat"), MessageType: reflect.TypeFor[testMessage]()}},
		}}
		if _, _, err := manual.Validate(Envelope{CampaignID: "camp-1", Message: testMessage{typ: "test.command", name: "ok"}}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})

	t.Run("success", func(t *testing.T) {
		envelope, spec, err := catalog.Validate(Envelope{CampaignID: "camp-1", Message: testMessage{typ: "test.command", name: " ok "}})
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if got, want := envelope.CampaignID, "camp-1"; got != want {
			t.Fatalf("campaign id = %q, want %q", got, want)
		}
		if got, want := spec.Definition().Type, Type("test.command"); got != want {
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

	t.Run("padded campaign id", func(t *testing.T) {
		if _, _, err := catalog.Validate(Envelope{CampaignID: " camp-1 ", Message: testMessage{typ: "test.command", name: "ok"}}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})

	t.Run("message normalization failure", func(t *testing.T) {
		manual := &Catalog{specs: map[Type]Spec{
			"test.command": stubSpec{
				def:          Definition{Type: "test.command", Owner: OwnerCore, Scope: ScopeCampaign, MessageType: reflect.TypeFor[testMessage]()},
				normalizeErr: errors.New("boom"),
			},
		}}
		if _, _, err := manual.Validate(Envelope{CampaignID: "camp-1", Message: testMessage{typ: "test.command", name: "ok"}}); err == nil {
			t.Fatal("Validate() error = nil, want failure")
		}
	})
}

func TestCatalogTypes(t *testing.T) {
	t.Parallel()

	var nilCatalog *Catalog
	if got := nilCatalog.Types(); got != nil {
		t.Fatalf("nil catalog Types() = %v, want nil", got)
	}

	first := NewCoreSpec(CoreSpecArgs[testMessage]{Message: testMessage{typ: "test.first"}, Scope: ScopeCampaign})
	second := NewCoreSpec(CoreSpecArgs[testMessage]{Message: testMessage{typ: "test.second"}, Scope: ScopeNewCampaign})
	catalog, err := NewCatalog(first, second)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	types := catalog.Types()
	if got, want := len(types), 2; got != want {
		t.Fatalf("Types() len = %d, want %d", got, want)
	}
	seen := make(map[Type]bool, len(types))
	for _, typ := range types {
		seen[typ] = true
	}
	if !seen[first.Definition().Type] || !seen[second.Definition().Type] {
		t.Fatalf("Types() = %v, want both registered types", types)
	}
}

type testMessage struct {
	typ  Type
	name string
}

func (m testMessage) CommandType() Type { return m.typ }

type otherMessage struct{}

func (otherMessage) CommandType() Type { return "other.command" }

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
