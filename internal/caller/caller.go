package caller

import (
	"fmt"

	"github.com/fracturing-space/game/internal/canonical"
)

// Kind labels the caller identity domain.
type Kind string

const (
	// KindSubject identifies an external authenticated user subject.
	KindSubject Kind = "subject"
	// KindAIAgent identifies one internal AI operator bound to a campaign.
	KindAIAgent Kind = "ai_agent"
)

// Caller is the authenticated or internal caller identity presented to the service.
type Caller struct {
	SubjectID string
	AIAgentID string
}

// NewSubject returns one validated subject caller identity.
func NewSubject(subjectID string) (Caller, error) {
	if canonical.IsBlank(subjectID) {
		return Caller{}, fmt.Errorf("subject id is required")
	}
	if !canonical.IsExact(subjectID) {
		return Caller{}, fmt.Errorf("subject id must not contain surrounding whitespace")
	}
	return Caller{SubjectID: subjectID}, nil
}

// NewAIAgent returns one validated ai-agent caller identity.
func NewAIAgent(aiAgentID string) (Caller, error) {
	if canonical.IsBlank(aiAgentID) {
		return Caller{}, fmt.Errorf("ai agent id is required")
	}
	if !canonical.IsExact(aiAgentID) {
		return Caller{}, fmt.Errorf("ai agent id must not contain surrounding whitespace")
	}
	return Caller{AIAgentID: aiAgentID}, nil
}

// MustNewSubject returns one validated subject caller identity or panics.
func MustNewSubject(subjectID string) Caller {
	next, err := NewSubject(subjectID)
	if err != nil {
		panic(err)
	}
	return next
}

// MustNewAIAgent returns one validated ai-agent caller identity or panics.
func MustNewAIAgent(aiAgentID string) Caller {
	next, err := NewAIAgent(aiAgentID)
	if err != nil {
		panic(err)
	}
	return next
}

// Valid reports whether the caller carries exactly one usable identity binding.
func (c Caller) Valid() bool {
	switch {
	case c.SubjectID != "" && c.AIAgentID == "":
		return true
	case c.SubjectID == "" && c.AIAgentID != "":
		return true
	default:
		return false
	}
}

// SameIdentity reports whether both callers carry the same identity binding.
func (c Caller) SameIdentity(other Caller) bool {
	return c.SubjectID == other.SubjectID && c.AIAgentID == other.AIAgentID
}

// Kind reports which identity domain the caller uses.
func (c Caller) Kind() Kind {
	switch {
	case c.SubjectID != "":
		return KindSubject
	case c.AIAgentID != "":
		return KindAIAgent
	default:
		return ""
	}
}
