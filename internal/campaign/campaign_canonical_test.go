package campaign

import "testing"

func TestCanonicalValidationBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{name: "ai bind padded", err: ValidateAIBind(AIBind{AIAgentID: " agent-1 "})},
		{name: "ai bound padded", err: ValidateAIBound(AIBound{AIAgentID: " agent-1 "})},
		{name: "play began padded session", err: ValidatePlayBegan(PlayBegan{SessionID: " sess-1 ", SceneID: "scene-1"})},
		{name: "play began padded scene", err: ValidatePlayBegan(PlayBegan{SessionID: "sess-1", SceneID: " scene-1 "})},
		{name: "play paused padded session", err: ValidatePlayPaused(PlayPaused{SessionID: " sess-1 ", SceneID: "scene-1", Reason: "pause"})},
		{name: "play paused padded scene", err: ValidatePlayPaused(PlayPaused{SessionID: "sess-1", SceneID: " scene-1 ", Reason: "pause"})},
		{name: "play resumed padded session", err: ValidatePlayResumed(PlayResumed{SessionID: " sess-1 ", SceneID: "scene-1", Reason: "resume"})},
		{name: "play resumed padded scene", err: ValidatePlayResumed(PlayResumed{SessionID: "sess-1", SceneID: " scene-1 ", Reason: "resume"})},
		{name: "play ended padded session", err: ValidatePlayEnded(PlayEnded{SessionID: " sess-1 "})},
		{name: "play ended padded scene", err: ValidatePlayEnded(PlayEnded{SessionID: "sess-1", SceneID: " scene-1 "})},
	}
	for _, test := range tests {
		if test.err == nil {
			t.Fatalf("%s error = nil, want failure", test.name)
		}
	}

	if err := ValidatePlayEnded(PlayEnded{SessionID: "sess-1"}); err != nil {
		t.Fatalf("ValidatePlayEnded(empty scene) error = %v", err)
	}
	if err := ValidatePlayEnded(PlayEnded{SessionID: "sess-1", SceneID: "scene-1"}); err != nil {
		t.Fatalf("ValidatePlayEnded(scene) error = %v", err)
	}
}
