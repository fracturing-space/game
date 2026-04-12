# Game Service

This document records the stable design intent for the current experimental
game service. It points to source contracts instead of duplicating transport or
generated shapes that would drift.

## Goal

The service is a small replay-first game runtime built around one authoritative
campaign timeline:

- commands express intent
- accepted commands emit immutable events
- events are the only state mutation path
- campaign state is rebuilt by replaying committed events in order

The repo is still experimental. Breaking changes to public and internal
contracts are acceptable while the command model is being shaped.

## Source Of Truth

The public gRPC surface is defined in:

- `api/proto/game/v1/service.proto`
- supporting messages under `api/proto/game/v1/*.proto`

Contributor-facing domain contracts live under:

- `internal/campaign`
- `internal/participant`
- `internal/character`
- `internal/session`
- `internal/scene`

Command and event behavior lives in the matching `internal/modules/...`
packages, and end-to-end orchestration lives in `internal/service`.

## Public Lifecycle

The public play lifecycle is intentionally small:

- `BeginPlay` is the public `SETUP -> ACTIVE` transition
- `BeginPlay` creates the active session if none exists
- `EndPlay` is the public `ACTIVE|PAUSED -> SETUP` transition

`session.started` and `session.ended` remain internal lifecycle events. They
record what happened inside the campaign timeline, but they are not separate
public lifecycle RPCs.

Scene-management and internal plan or execute-plan flows are not part of the
current public owner-facing gRPC surface. Those concepts remain internal until
there is a real AI operator path for GM command routing.

## Core Invariants

- events are the only mutation path
- each accepted command becomes one atomic commit batch of one or more events
- replay is deterministic and authoritative
- internal command and event logic must not depend on wall-clock time
- stored timestamps are journal metadata only and do not affect fold
- any non-empty `subject_id` may be bound at most once per campaign
- IDs are never reused; deletion is modeled as inactive event-backed state

## Authorization And Readiness

Every RPC requires trusted `x-fs-subject-id` metadata.

Current owner-facing rules:

- any authenticated caller may create a campaign
- `CreateCampaign` also creates:
  - one bound owner participant for the caller
- only the owner may manage campaign metadata, participants, characters, play
  lifecycle, and campaign AI binding
- only bound participants may read campaign state or persisted events

Entering play currently requires:

- a bound campaign `ai_agent_id`
- every active participant to have a bound `subject_id`
- at least one active PC for each bound participant

Missing session and missing scene are not separate readiness blockers because
`BeginPlay` can create the session automatically and the current public API
does not expose scene management.

## Randomness

Game-affecting randomness must be replay-safe.

- direct runtime randomness must not affect fold or decision results
- random outcomes that matter to the game should be derived from deterministic,
  event-backed state
- if campaign randomness is introduced later, it should carry explicit state
  such as a seed and step or counter so replay remains authoritative

## Package Boundaries

- `internal/caller`, `internal/command`, and `internal/event` provide shared
  typed infrastructure
- `internal/authz`, `internal/admission`, and `internal/readiness` provide
  reusable policy
- `internal/service` loads, checks, replays, and executes the campaign aggregate
- `internal/transport/grpc/gamev1` maps protobuf RPCs to the service layer
- `cmd/server` wires the local development server

For most changes, the fastest reading order is:

1. the relevant contract package in `internal/...`
2. the matching `internal/modules/...` package
3. `internal/service`
4. `internal/transport/grpc/gamev1` if the change touches RPC shape
