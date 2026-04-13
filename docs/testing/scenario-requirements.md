# Scenario Design

This document defines the scenario model the repo should build toward. It does
not add a runner in this pass, but it is now concrete enough that another
engineer can implement helpers or a runner without inventing new concepts.

## Current Decision

The repo has enough lifecycle coverage to move beyond a requirements-only note.
Scenarios should now be designed around the current public gRPC surface and the
current campaign lifecycle contract:

- campaign state exposes `play_state`, not `mode`
- the public lifecycle is currently `SETUP -> ACTIVE -> SETUP`
- `PAUSED` exists in the model but is not yet a public gRPC scenario step
- `campaign.created` does not carry lifecycle state

The first implementation path should remain Go-based and transport-real. If the
repo later adds file-backed scenarios, they must serialize the same scenario
model rather than introduce a separate DSL with different semantics.

## Scenario Model

Scenarios should use one small declarative core model:

- `step`: one ordered action, usually one gRPC call
- `caller`: the trusted subject identity attached to that step
- `target`: whether the step runs as live execution, plan, or execute-plan
- `ref`: a symbolic handle bound to one identifier returned by an earlier step
- `assert`: the expected response, projected result, committed state, event
  batch, or structured failure

That model should be usable in two authoring forms without changing meaning:

- Go helpers or builders for contributors who want code-native scenarios
- file-backed scenarios later, if they prove worth the extra machinery

The important constraint is that Go remains a first-class authoring path. The
repo should not reintroduce a separate language that contributors must learn
just to describe system flows.

## Required Capabilities

The scenario system should support:

- human-readable ordered multi-step sequences
- execution through the real gRPC transport
- trusted caller metadata on every step
- symbolic ID capture and reuse across later steps
- live execution plus plan or execute-plan execution
- assertions on response fields and returned IDs
- assertions on projected planned commits or events
- assertions on committed campaign state, including `play_state`
- negative-case assertions for rejected calls
- assertions on structured readiness failures
- committed event-stream assertions for ordered lifecycle events

Replay or restart-oriented behavior can stay deferred until the runtime grows
enough to need it.

## Canonical Scenario Families

The first concrete scenario suite should cover these flows:

- `Lifecycle Happy Path`
  Create campaign, bind AI, create player character, start session, create and
  activate scene, verify readiness, begin play, end session, and assert
  committed `play_state` and event order.
- `Readiness Gating`
  Attempt `BeginPlay` before prerequisites exist, assert structured
  `FailedPrecondition` details, satisfy blockers incrementally, then assert the
  successful transition to `ACTIVE`.
- `Participant Binding And Visibility`
  Assert that an unjoined caller is denied, an unbound participant is still
  denied, a bound participant unlocks read visibility, and list visibility
  changes with the binding.
- `Plan And Execute`
  Plan one or more existing-campaign mutations, assert projected events and
  projected `play_state`, then execute the plan and assert the committed state
  matches the projection.
- `Committed Stream`
  Subscribe to campaign events and assert ordered committed lifecycle types from
  create through begin-play and end-session.

## Current Boundaries

The design should optimize for the lifecycle the service already exposes. It
should explicitly avoid stretching around missing interaction features.

This pass does not require:

- GM-authored interaction flow
- player response or turn sequencing
- public pause or resume RPCs
- a standalone runner binary
- Lua or another general scripting language
- Makefile or CI scenario targets beyond normal Go test execution

When pause/resume become public, they should slot into the same model as normal
steps and assertions; no scenario redesign should be needed.
