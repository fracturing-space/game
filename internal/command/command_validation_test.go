package command

import (
	"reflect"
	"testing"
)

func TestValidateDefinitionRejectsPaddedCanonicalFields(t *testing.T) {
	t.Parallel()

	messageType := reflect.TypeFor[testMessage]()

	if err := validateDefinition(Definition{
		Type:        " test.command ",
		Owner:       OwnerCore,
		Scope:       ScopeCampaign,
		MessageType: messageType,
	}); err == nil {
		t.Fatal("validateDefinition(padded type) error = nil, want failure")
	}

	if err := validateDefinition(Definition{
		Type:        "sys.demo.command",
		Owner:       OwnerSystem,
		SystemID:    " demo ",
		Scope:       ScopeCampaign,
		MessageType: messageType,
	}); err == nil {
		t.Fatal("validateDefinition(padded system id) error = nil, want failure")
	}
}
