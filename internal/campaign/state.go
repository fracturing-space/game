package campaign

import (
	"maps"
	"slices"
	"strings"

	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

// PlayState declares the current campaign play lifecycle state.
type PlayState string

const (
	// PlayStateSetup allows setup-time modifications outside active play.
	PlayStateSetup PlayState = "SETUP"
	// PlayStateActive allows in-game actions and GM planning.
	PlayStateActive PlayState = "ACTIVE"
	// PlayStatePaused pauses active play for out-of-character coordination.
	PlayStatePaused PlayState = "PAUSED"
)

// State is the replayed campaign state.
type State struct {
	Exists          bool
	CampaignID      string
	Name            string
	PlayState       PlayState
	AIAgentID       string
	SessionCount    uint64
	ActiveSessionID string
	ActiveSceneID   string
	Characters      map[string]character.Record
	Participants    map[string]participant.Record
	Scenes          map[string]scene.Record
	Sessions        map[string]session.Record
}

// Snapshot is the stable public read shape for the current slice.
type Snapshot struct {
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	PlayState       PlayState            `json:"play_state"`
	AIAgentID       string               `json:"ai_agent_id"`
	ActiveSessionID string               `json:"active_session_id,omitempty"`
	ActiveSceneID   string               `json:"active_scene_id,omitempty"`
	Characters      []character.Record   `json:"characters"`
	Participants    []participant.Record `json:"participants"`
	Scenes          []scene.Record       `json:"scenes"`
	Sessions        []session.Record     `json:"sessions"`
}

// NewState returns an initialized zero-value campaign state.
func NewState() State {
	return State{
		Characters:   make(map[string]character.Record),
		Participants: make(map[string]participant.Record),
		Scenes:       make(map[string]scene.Record),
		Sessions:     make(map[string]session.Record),
	}
}

// Clone returns a deep copy of the campaign state.
func (s State) Clone() State {
	clone := s
	clone.Characters = make(map[string]character.Record, len(s.Characters))
	maps.Copy(clone.Characters, s.Characters)
	clone.Participants = make(map[string]participant.Record, len(s.Participants))
	maps.Copy(clone.Participants, s.Participants)
	clone.Scenes = make(map[string]scene.Record, len(s.Scenes))
	for sceneID, record := range s.Scenes {
		clone.Scenes[sceneID] = cloneSceneRecord(record)
	}
	clone.Sessions = make(map[string]session.Record, len(s.Sessions))
	for sessionID, record := range s.Sessions {
		clone.Sessions[sessionID] = cloneSessionRecord(record)
	}
	return clone
}

// SnapshotOf returns the stable public shape for the provided state.
func SnapshotOf(state State) Snapshot {
	characters := make([]character.Record, 0, len(state.Characters))
	for _, record := range state.Characters {
		if !record.Active {
			continue
		}
		characters = append(characters, record)
	}
	slices.SortFunc(characters, func(a, b character.Record) int {
		return strings.Compare(a.ID, b.ID)
	})

	participants := make([]participant.Record, 0, len(state.Participants))
	for _, record := range state.Participants {
		if !record.Active {
			continue
		}
		participants = append(participants, record)
	}
	slices.SortFunc(participants, func(a, b participant.Record) int {
		return strings.Compare(a.ID, b.ID)
	})

	scenes := make([]scene.Record, 0, len(state.Scenes))
	for _, record := range state.Scenes {
		scenes = append(scenes, cloneSceneRecord(record))
	}
	slices.SortFunc(scenes, func(a, b scene.Record) int {
		return strings.Compare(a.ID, b.ID)
	})

	sessions := make([]session.Record, 0, len(state.Sessions))
	for _, record := range state.Sessions {
		sessions = append(sessions, cloneSessionRecord(record))
	}
	slices.SortFunc(sessions, func(a, b session.Record) int {
		return strings.Compare(a.ID, b.ID)
	})

	return Snapshot{
		ID:              state.CampaignID,
		Name:            state.Name,
		PlayState:       state.PlayState,
		AIAgentID:       state.AIAgentID,
		ActiveSessionID: state.ActiveSessionID,
		ActiveSceneID:   state.ActiveSceneID,
		Characters:      characters,
		Participants:    participants,
		Scenes:          scenes,
		Sessions:        sessions,
	}
}

// ActiveSession returns the active session snapshot when one is present.
func (s State) ActiveSession() *session.Record {
	if s.ActiveSessionID == "" {
		return nil
	}
	record, ok := s.Sessions[s.ActiveSessionID]
	if !ok {
		return nil
	}
	return session.CloneRecord(&record)
}

// ActiveSession returns the active session snapshot when one is present.
func (s Snapshot) ActiveSession() *session.Record {
	return s.Session(s.ActiveSessionID)
}

// Session returns the requested session snapshot when one is present.
func (s Snapshot) Session(sessionID string) *session.Record {
	if sessionID == "" {
		return nil
	}
	for _, next := range s.Sessions {
		if next.ID != sessionID {
			continue
		}
		copy := cloneSessionRecord(next)
		return &copy
	}
	return nil
}

// ActiveScene returns the active scene snapshot when one is present.
func (s Snapshot) ActiveScene() *scene.Record {
	if s.ActiveSceneID == "" {
		return nil
	}
	for _, next := range s.Scenes {
		if next.ID != s.ActiveSceneID {
			continue
		}
		copy := cloneSceneRecord(next)
		return &copy
	}
	return nil
}

// ScenesForSession returns clone-safe scenes for one session id.
func (s Snapshot) ScenesForSession(sessionID string) []scene.Record {
	if sessionID == "" {
		return nil
	}
	scenes := make([]scene.Record, 0, len(s.Scenes))
	for _, next := range s.Scenes {
		if next.SessionID != sessionID {
			continue
		}
		scenes = append(scenes, cloneSceneRecord(next))
	}
	return scenes
}

func cloneSceneRecord(record scene.Record) scene.Record {
	record.CharacterIDs = append([]string(nil), record.CharacterIDs...)
	return record
}

func cloneSessionRecord(record session.Record) session.Record {
	if cloned := session.CloneRecord(&record); cloned != nil {
		return *cloned
	}
	return session.Record{}
}

// Valid reports whether the play-state value is recognized.
func (s PlayState) Valid() bool {
	switch s {
	case PlayStateSetup, PlayStateActive, PlayStatePaused:
		return true
	default:
		return false
	}
}
