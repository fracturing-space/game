package gamev1

import (
	"context"
	"strings"
	"testing"

	gamev1pb "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/service"
)

type scenarioSpec struct {
	name       string
	newHarness func(*testing.T) scenarioHarness
	steps      []scenarioStep
}

type scenarioStep struct {
	name   string
	caller string
	action func(context.Context, *scenarioRuntime) (any, error)
	assert func(*testing.T, *scenarioRuntime, any)
}

type scenarioRuntime struct {
	baseT   *testing.T
	t       *testing.T
	harness scenarioHarness
	callers map[string]context.Context
	refs    map[string]string
	streams map[string]scenarioStream
}

type scenarioStream struct {
	client gamev1pb.GameService_StreamCampaignEventsClient
	cancel context.CancelFunc
}

func runScenario(t *testing.T, spec scenarioSpec) {
	t.Helper()

	if strings.TrimSpace(spec.name) == "" {
		t.Fatal("scenario name is required")
	}

	t.Run(spec.name, func(t *testing.T) {
		runtime := newScenarioRuntime(t, spec.newHarness)
		for _, step := range spec.steps {
			t.Run(step.name, func(t *testing.T) {
				t.Helper()

				runtime.t = t
				result, err := step.action(runtime.callerContext(step.caller), runtime)
				if err != nil {
					t.Fatalf("%s error = %v", step.name, err)
				}
				if step.assert != nil {
					step.assert(t, runtime, result)
				}
			})
		}
	})
}

func newScenarioRuntime(t *testing.T, newHarness func(*testing.T) scenarioHarness) *scenarioRuntime {
	t.Helper()

	if newHarness == nil {
		newHarness = newScenarioHarness
	}
	runtime := &scenarioRuntime{
		baseT:   t,
		t:       t,
		harness: newHarness(t),
		callers: make(map[string]context.Context),
		refs:    make(map[string]string),
		streams: make(map[string]scenarioStream),
	}
	t.Cleanup(runtime.closeStreams)
	t.Cleanup(runtime.closeHarness)
	return runtime
}

func (r *scenarioRuntime) callerContext(caller string) context.Context {
	r.t.Helper()

	caller = strings.TrimSpace(caller)
	if caller == "" {
		r.t.Fatal("scenario caller is required")
	}
	if ctx, ok := r.callers[caller]; ok {
		return ctx
	}
	ctx := outgoingSubjectContext(caller)
	r.callers[caller] = ctx
	return ctx
}

func (r *scenarioRuntime) setRef(name string, value string) {
	r.t.Helper()

	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)
	if name == "" {
		r.t.Fatal("scenario ref name is required")
	}
	if value == "" {
		r.t.Fatalf("scenario ref %q value is empty", name)
	}
	r.refs[name] = value
}

func (r *scenarioRuntime) ref(name string) string {
	r.t.Helper()

	name = strings.TrimSpace(name)
	value, ok := r.refs[name]
	if !ok || value == "" {
		r.t.Fatalf("scenario ref %q is not set", name)
	}
	return value
}

func (r *scenarioRuntime) setStream(name string, stream scenarioStream) {
	r.t.Helper()

	name = strings.TrimSpace(name)
	if name == "" {
		r.t.Fatal("scenario stream name is required")
	}
	r.streams[name] = stream
}

func (r *scenarioRuntime) stream(name string) scenarioStream {
	r.t.Helper()

	name = strings.TrimSpace(name)
	stream, ok := r.streams[name]
	if !ok || stream.client == nil {
		r.t.Fatalf("scenario stream %q is not set", name)
	}
	return stream
}

func (r *scenarioRuntime) closeStreams() {
	for _, stream := range r.streams {
		if stream.cancel != nil {
			stream.cancel()
		}
	}
	clear(r.streams)
}

func (r *scenarioRuntime) closeHarness() {
	if r.harness.close != nil {
		r.harness.close()
	}
}

func (r *scenarioRuntime) restartHarness() {
	r.t.Helper()

	reopen := r.harness.reopen
	if reopen == nil {
		r.t.Fatal("scenario harness restart is not supported")
	}
	r.closeStreams()
	r.closeHarness()
	r.harness = reopen(r.baseT)
}

func (r *scenarioRuntime) projections() service.ProjectionStore {
	r.t.Helper()

	if r.harness.projections == nil {
		r.t.Fatal("scenario harness does not expose a projection store")
	}
	return r.harness.projections
}

func scenarioResultAs[T any](t *testing.T, got any) T {
	t.Helper()

	typed, ok := got.(T)
	if !ok {
		t.Fatalf("scenario result type = %T, want %T", got, *new(T))
	}
	return typed
}
