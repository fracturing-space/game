package event

import (
	"fmt"
	"reflect"
	"time"

	"github.com/fracturing-space/game/internal/canonical"
)

// Type identifies one stable event contract.
type Type string

// Owner identifies whether an event belongs to the core or a registered
// system.
type Owner string

const (
	// OwnerCore identifies core-owned event contracts.
	OwnerCore Owner = "core"
	// OwnerSystem identifies system-owned event contracts.
	OwnerSystem Owner = "system"
)

// Message is the typed event payload.
type Message interface {
	EventType() Type
}

// Envelope is one typed event instance.
type Envelope struct {
	CampaignID string
	Message    Message
}

// Type returns the normalized event type for the enclosed message.
func (e Envelope) Type() Type {
	if e.Message == nil {
		return ""
	}
	return e.Message.EventType()
}

// Record is one persisted timeline event with its sequence number.
type Record struct {
	Seq        uint64
	CommitSeq  uint64
	RecordedAt time.Time
	Envelope
}

// Definition describes one event type.
type Definition struct {
	Type        Type
	Owner       Owner
	SystemID    string
	MessageType reflect.Type
}

// Spec is the registration surface for one event type.
type Spec interface {
	Definition() Definition
	NormalizeMessage(Message) (Message, error)
	ValidateMessage(Message) error
}

// TypedSpec keeps message typing at the definition source.
type TypedSpec[M Message] struct {
	definition Definition
	normalize  func(M) M
	validate   func(M) error
}

// NewCoreSpec constructs one core event spec.
func NewCoreSpec[M Message](message M, normalize func(M) M, validate func(M) error) TypedSpec[M] {
	return newTypedSpec(message, Definition{
		Type:        message.EventType(),
		Owner:       OwnerCore,
		MessageType: reflect.TypeOf(message),
	}, normalize, validate)
}

// NewSystemSpec constructs one system-owned event spec.
func NewSystemSpec[M Message](message M, systemID string, normalize func(M) M, validate func(M) error) TypedSpec[M] {
	return newTypedSpec(message, Definition{
		Type:        message.EventType(),
		Owner:       OwnerSystem,
		SystemID:    systemID,
		MessageType: reflect.TypeOf(message),
	}, normalize, validate)
}

func newTypedSpec[M Message](message M, definition Definition, normalize func(M) M, validate func(M) error) TypedSpec[M] {
	definition.Type = message.EventType()
	definition.MessageType = reflect.TypeOf(message)
	if normalize == nil {
		panic("event normalizer is required")
	}
	return TypedSpec[M]{
		definition: definition,
		normalize:  normalize,
		validate:   validate,
	}
}

// Definition returns the non-generic event metadata.
func (s TypedSpec[M]) Definition() Definition {
	return s.definition
}

// NormalizeMessage canonicalizes one message instance for this spec.
func (s TypedSpec[M]) NormalizeMessage(message Message) (Message, error) {
	typed, ok := message.(M)
	if !ok {
		return nil, fmt.Errorf("event %s must carry %v, got %T", s.definition.Type, s.definition.MessageType, message)
	}
	return s.normalize(typed), nil
}

// Identity is an explicit no-op normalizer for messages with no canonicalization needs.
func Identity[M Message](message M) M {
	return message
}

// ValidateMessage checks typing and message-specific validation.
func (s TypedSpec[M]) ValidateMessage(message Message) error {
	typed, ok := message.(M)
	if !ok {
		return fmt.Errorf("event %s must carry %v, got %T", s.definition.Type, s.definition.MessageType, message)
	}
	if s.validate == nil {
		return nil
	}
	return s.validate(typed)
}

// NewEnvelope constructs one validated event envelope.
func NewEnvelope[M Message](spec TypedSpec[M], campaignID string, message M) (Envelope, error) {
	envelope := Envelope{
		CampaignID: campaignID,
		Message:    message,
	}
	definition := spec.Definition()
	if err := validateDefinition(definition); err != nil {
		return Envelope{}, err
	}
	if envelope.CampaignID == "" {
		return Envelope{}, fmt.Errorf("event campaign id is required")
	}
	if !canonical.IsExact(envelope.CampaignID) {
		return Envelope{}, fmt.Errorf("event campaign id must not contain surrounding whitespace")
	}
	if envelope.Message == nil {
		return Envelope{}, fmt.Errorf("event message is required")
	}
	envelope.Message = spec.normalize(message)
	if envelope.Type() != definition.Type {
		return Envelope{}, fmt.Errorf("event type is not registered: %s", envelope.Type())
	}
	if err := spec.ValidateMessage(envelope.Message); err != nil {
		return Envelope{}, err
	}
	return envelope, nil
}

// MessageAs returns the enclosed message as the requested concrete type.
func MessageAs[M Message](envelope Envelope) (M, error) {
	message, ok := envelope.Message.(M)
	if !ok {
		var zero M
		return zero, fmt.Errorf("event %s must carry %T, got %T", envelope.Type(), zero, envelope.Message)
	}
	return message, nil
}

// Catalog validates event envelopes.
type Catalog struct {
	specs map[Type]Spec
}

// NewCatalog constructs a validated event catalog.
func NewCatalog(specs ...Spec) (*Catalog, error) {
	catalog := &Catalog{specs: make(map[Type]Spec, len(specs))}
	for _, spec := range specs {
		if spec == nil {
			return nil, fmt.Errorf("event spec is required")
		}
		definition := spec.Definition()
		if err := validateDefinition(definition); err != nil {
			return nil, err
		}
		if _, exists := catalog.specs[definition.Type]; exists {
			return nil, fmt.Errorf("event type already registered: %s", definition.Type)
		}
		catalog.specs[definition.Type] = spec
	}
	return catalog, nil
}

// Validate checks an envelope against the catalog and returns the normalized
// envelope with the matched spec.
func (c *Catalog) Validate(envelope Envelope) (Envelope, Spec, error) {
	if c == nil {
		return Envelope{}, nil, fmt.Errorf("event catalog is required")
	}
	if envelope.CampaignID == "" {
		return Envelope{}, nil, fmt.Errorf("event campaign id is required")
	}
	if !canonical.IsExact(envelope.CampaignID) {
		return Envelope{}, nil, fmt.Errorf("event campaign id must not contain surrounding whitespace")
	}
	if envelope.Message == nil {
		return Envelope{}, nil, fmt.Errorf("event message is required")
	}

	spec, ok := c.specs[envelope.Type()]
	if !ok {
		return Envelope{}, nil, fmt.Errorf("event type is not registered: %s", envelope.Type())
	}
	normalizedMessage, err := spec.NormalizeMessage(envelope.Message)
	if err != nil {
		return Envelope{}, nil, err
	}
	envelope.Message = normalizedMessage
	if err := spec.ValidateMessage(envelope.Message); err != nil {
		return Envelope{}, nil, err
	}
	return envelope, spec, nil
}

// SpecFor returns the registered spec for one event type.
func (c *Catalog) SpecFor(eventType Type) (Spec, bool) {
	if c == nil {
		return nil, false
	}
	spec, ok := c.specs[eventType]
	return spec, ok
}

func validateDefinition(definition Definition) error {
	if err := canonical.ValidateOwnedType("event", string(definition.Type), definition.SystemID, definition.Owner == OwnerSystem, fmt.Errorf); err != nil {
		return err
	}
	if definition.MessageType == nil {
		return fmt.Errorf("event %s message type is required", definition.Type)
	}
	switch definition.Owner {
	case OwnerCore, OwnerSystem:
	default:
		return fmt.Errorf("event %s owner is invalid", definition.Type)
	}
	return nil
}
