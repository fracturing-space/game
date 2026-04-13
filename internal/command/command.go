package command

import (
	"fmt"
	"reflect"

	"github.com/fracturing-space/game/internal/canonical"
)

// Type identifies one stable command contract.
type Type string

// Owner identifies whether a command belongs to the core or a registered
// system.
type Owner string

const (
	// OwnerCore identifies core-owned command contracts.
	OwnerCore Owner = "core"
	// OwnerSystem identifies system-owned command contracts.
	OwnerSystem Owner = "system"
)

// Scope declares whether a command targets an existing campaign or creates a
// new one.
type Scope string

const (
	// ScopeCampaign requires a concrete campaign id on the command envelope.
	ScopeCampaign Scope = "campaign"
	// ScopeNewCampaign requires an empty campaign id because the command creates
	// the campaign itself.
	ScopeNewCampaign Scope = "new_campaign"
)

// Message is the typed command payload.
type Message interface {
	CommandType() Type
}

// Envelope is one typed command instance.
type Envelope struct {
	CampaignID string
	Message    Message
}

// Type returns the normalized command type for the enclosed message.
func (e Envelope) Type() Type {
	if e.Message == nil {
		return ""
	}
	return e.Message.CommandType()
}

// Definition describes one command type.
type Definition struct {
	Type        Type
	Owner       Owner
	Scope       Scope
	SystemID    string
	MessageType reflect.Type
}

// Spec is the registration surface for one command type.
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

// CoreSpecArgs configures one core-owned command spec.
type CoreSpecArgs[M Message] struct {
	Message   M
	Scope     Scope
	Normalize func(M) M
	Validate  func(M) error
}

// SystemSpecArgs configures one system-owned command spec.
type SystemSpecArgs[M Message] struct {
	Message   M
	SystemID  string
	Scope     Scope
	Normalize func(M) M
	Validate  func(M) error
}

// NewCoreSpec constructs one core command spec.
func NewCoreSpec[M Message](args CoreSpecArgs[M]) TypedSpec[M] {
	return newTypedSpec(args.Message, Definition{
		Type:        args.Message.CommandType(),
		Owner:       OwnerCore,
		Scope:       args.Scope,
		MessageType: reflect.TypeOf(args.Message),
	}, args.Normalize, args.Validate)
}

// NewSystemSpec constructs one system-owned command spec.
func NewSystemSpec[M Message](args SystemSpecArgs[M]) TypedSpec[M] {
	return newTypedSpec(args.Message, Definition{
		Type:        args.Message.CommandType(),
		Owner:       OwnerSystem,
		Scope:       args.Scope,
		SystemID:    args.SystemID,
		MessageType: reflect.TypeOf(args.Message),
	}, args.Normalize, args.Validate)
}

func newTypedSpec[M Message](message M, definition Definition, normalize func(M) M, validate func(M) error) TypedSpec[M] {
	definition.Type = message.CommandType()
	definition.MessageType = reflect.TypeOf(message)
	if normalize == nil {
		normalize = Identity[M]
	}
	return TypedSpec[M]{
		definition: definition,
		normalize:  normalize,
		validate:   validate,
	}
}

// Definition returns the non-generic command metadata.
func (s TypedSpec[M]) Definition() Definition {
	return s.definition
}

// NormalizeMessage canonicalizes one message instance for this spec.
func (s TypedSpec[M]) NormalizeMessage(message Message) (Message, error) {
	typed, ok := message.(M)
	if !ok {
		return nil, fmt.Errorf("command %s must carry %v, got %T", s.definition.Type, s.definition.MessageType, message)
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
		return fmt.Errorf("command %s must carry %v, got %T", s.definition.Type, s.definition.MessageType, message)
	}
	if s.validate == nil {
		return nil
	}
	return s.validate(typed)
}

// MessageAs returns the enclosed message as the requested concrete type.
func MessageAs[M Message](envelope Envelope) (M, error) {
	message, ok := envelope.Message.(M)
	if !ok {
		var zero M
		return zero, fmt.Errorf("command %s must carry %T, got %T", envelope.Type(), zero, envelope.Message)
	}
	return message, nil
}

// Catalog validates command envelopes.
type Catalog struct {
	specs map[Type]Spec
}

// NewCatalog constructs a validated command catalog.
func NewCatalog(specs ...Spec) (*Catalog, error) {
	catalog := &Catalog{specs: make(map[Type]Spec, len(specs))}
	for _, spec := range specs {
		if spec == nil {
			return nil, fmt.Errorf("command spec is required")
		}
		definition := spec.Definition()
		if err := validateDefinition(definition); err != nil {
			return nil, err
		}
		if _, exists := catalog.specs[definition.Type]; exists {
			return nil, fmt.Errorf("command type already registered: %s", definition.Type)
		}
		catalog.specs[definition.Type] = spec
	}
	return catalog, nil
}

// Validate checks an envelope against the catalog and returns the normalized
// envelope with the matched spec.
func (c *Catalog) Validate(envelope Envelope) (Envelope, Spec, error) {
	if c == nil {
		return Envelope{}, nil, fmt.Errorf("command catalog is required")
	}
	if envelope.Message == nil {
		return Envelope{}, nil, fmt.Errorf("command message is required")
	}
	if envelope.CampaignID != "" && !canonical.IsExact(envelope.CampaignID) {
		return Envelope{}, nil, fmt.Errorf("command campaign id must not contain surrounding whitespace")
	}

	spec, ok := c.specs[envelope.Type()]
	if !ok {
		return Envelope{}, nil, fmt.Errorf("command type is not registered: %s", envelope.Type())
	}
	normalizedMessage, err := spec.NormalizeMessage(envelope.Message)
	if err != nil {
		return Envelope{}, nil, err
	}
	envelope.Message = normalizedMessage
	definition := spec.Definition()
	switch definition.Scope {
	case ScopeCampaign:
		if envelope.CampaignID == "" {
			return Envelope{}, nil, fmt.Errorf("campaign id is required for %s", definition.Type)
		}
	case ScopeNewCampaign:
		if envelope.CampaignID != "" {
			return Envelope{}, nil, fmt.Errorf("campaign id must be empty for %s", definition.Type)
		}
	default:
		return Envelope{}, nil, fmt.Errorf("command %s scope is invalid", definition.Type)
	}
	if err := spec.ValidateMessage(envelope.Message); err != nil {
		return Envelope{}, nil, err
	}
	return envelope, spec, nil
}

// Types returns the normalized command types registered in the catalog.
func (c *Catalog) Types() []Type {
	if c == nil {
		return nil
	}
	types := make([]Type, 0, len(c.specs))
	for typ := range c.specs {
		types = append(types, typ)
	}
	return types
}

func validateDefinition(definition Definition) error {
	if err := canonical.ValidateOwnedType("command", string(definition.Type), definition.SystemID, definition.Owner == OwnerSystem, fmt.Errorf); err != nil {
		return err
	}
	if definition.MessageType == nil {
		return fmt.Errorf("command %s message type is required", definition.Type)
	}
	switch definition.Owner {
	case OwnerCore, OwnerSystem:
	default:
		return fmt.Errorf("command %s owner is invalid", definition.Type)
	}
	switch definition.Scope {
	case ScopeCampaign, ScopeNewCampaign:
	default:
		return fmt.Errorf("command %s scope is invalid", definition.Type)
	}
	return nil
}
