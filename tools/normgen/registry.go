package main

type packageSpec struct {
	Dir       string
	Output    string
	Functions []functionSpec
}

type functionMode string

const (
	functionModeFinal functionMode = "final"
	functionModeBase  functionMode = "base"
)

type functionSpec struct {
	Name     string
	TypeName string
	Mode     functionMode
	Ops      []fieldOp
}

type fieldOpKind string

const (
	fieldOpTrimString          fieldOpKind = "trim_string"
	fieldOpTrimStringCast      fieldOpKind = "trim_string_cast"
	fieldOpNormalizeStringList fieldOpKind = "normalize_string_slice"
	fieldOpCall                fieldOpKind = "call"
)

type fieldOp struct {
	Kind      fieldOpKind
	FieldName string
	Helper    string
	Trim      bool
	DropEmpty bool
	Unique    bool
	Sort      bool
}

func registry() []packageSpec {
	return []packageSpec{
		{
			Dir:    "internal/campaign",
			Output: "internal/campaign/zz_normalize.go",
			Functions: []functionSpec{
				{Name: "normalizeCreate", TypeName: "Create", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}, {Kind: fieldOpTrimString, FieldName: "OwnerName"}}},
				{Name: "normalizeUpdate", TypeName: "Update", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}}},
				{Name: "normalizeAIBind", TypeName: "AIBind", Mode: functionModeFinal},
				{Name: "normalizePlayPause", TypeName: "PlayPause", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Reason"}}},
				{Name: "normalizePlayResume", TypeName: "PlayResume", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Reason"}}},
				{Name: "normalizeCreated", TypeName: "Created", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}}},
				{Name: "normalizeUpdated", TypeName: "Updated", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}}},
				{Name: "normalizeAIBound", TypeName: "AIBound", Mode: functionModeFinal},
				{Name: "normalizePlayBegan", TypeName: "PlayBegan", Mode: functionModeFinal},
				{Name: "normalizePlayPaused", TypeName: "PlayPaused", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Reason"}}},
				{Name: "normalizePlayResumed", TypeName: "PlayResumed", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Reason"}}},
				{Name: "normalizePlayEnded", TypeName: "PlayEnded", Mode: functionModeFinal},
			},
		},
		{
			Dir:    "internal/session",
			Output: "internal/session/zz_normalize.go",
			Functions: []functionSpec{
				{Name: "normalizeStart", TypeName: "Start", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}, {Kind: fieldOpCall, FieldName: "CharacterControllers", Helper: "CloneAssignments"}}},
				{Name: "normalizeStarted", TypeName: "Started", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}, {Kind: fieldOpCall, FieldName: "CharacterControllers", Helper: "CloneAssignments"}}},
				{Name: "normalizeEnded", TypeName: "Ended", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}, {Kind: fieldOpCall, FieldName: "CharacterControllers", Helper: "CloneAssignments"}}},
			},
		},
		{
			Dir:    "internal/scene",
			Output: "internal/scene/zz_normalize.go",
			Functions: []functionSpec{
				{Name: "normalizeCreate", TypeName: "Create", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}, {Kind: fieldOpCall, FieldName: "CharacterIDs", Helper: "CloneCharacterIDs"}}},
				{Name: "normalizeActivate", TypeName: "Activate", Mode: functionModeFinal},
				{Name: "normalizeEnd", TypeName: "End", Mode: functionModeFinal},
				{Name: "normalizeReplaceCast", TypeName: "ReplaceCast", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpCall, FieldName: "CharacterIDs", Helper: "CloneCharacterIDs"}}},
				{Name: "normalizeCreated", TypeName: "Created", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}, {Kind: fieldOpCall, FieldName: "CharacterIDs", Helper: "CloneCharacterIDs"}}},
				{Name: "normalizeActivated", TypeName: "Activated", Mode: functionModeFinal},
				{Name: "normalizeEnded", TypeName: "Ended", Mode: functionModeFinal},
				{Name: "normalizeCastReplacedEvent", TypeName: "CastReplaced", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpCall, FieldName: "CharacterIDs", Helper: "CloneCharacterIDs"}}},
			},
		},
		{
			Dir:    "internal/participant",
			Output: "internal/participant/zz_normalize.go",
			Functions: []functionSpec{
				{Name: "normalizeJoinBase", TypeName: "Join", Mode: functionModeBase, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}}},
				{Name: "normalizeUpdate", TypeName: "Update", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}}},
				{Name: "normalizeBind", TypeName: "Bind", Mode: functionModeFinal},
				{Name: "normalizeUnbind", TypeName: "Unbind", Mode: functionModeFinal},
				{Name: "normalizeLeave", TypeName: "Leave", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Reason"}}},
				{Name: "normalizeJoined", TypeName: "Joined", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}}},
				{Name: "normalizeUpdated", TypeName: "Updated", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}}},
				{Name: "normalizeBound", TypeName: "Bound", Mode: functionModeFinal},
				{Name: "normalizeUnbound", TypeName: "Unbound", Mode: functionModeFinal},
				{Name: "normalizeLeft", TypeName: "Left", Mode: functionModeFinal},
			},
		},
		{
			Dir:    "internal/character",
			Output: "internal/character/zz_normalize.go",
			Functions: []functionSpec{
				{Name: "normalizeCreateBase", TypeName: "Create", Mode: functionModeBase, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}}},
				{Name: "normalizeUpdateBase", TypeName: "Update", Mode: functionModeBase, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}}},
				{Name: "normalizeDelete", TypeName: "Delete", Mode: functionModeFinal, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Reason"}}},
				{Name: "normalizeCreatedBase", TypeName: "Created", Mode: functionModeBase, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}}},
				{Name: "normalizeUpdatedBase", TypeName: "Updated", Mode: functionModeBase, Ops: []fieldOp{{Kind: fieldOpTrimString, FieldName: "Name"}}},
				{Name: "normalizeDeleted", TypeName: "Deleted", Mode: functionModeFinal},
			},
		},
	}
}
