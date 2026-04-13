package gamev1

import (
	"context"
	"testing"

	gamev1pb "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/participant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRequestAndCallerHelpers(t *testing.T) {
	t.Parallel()

	createEnvelope := createCampaignEnvelope(&gamev1pb.CreateCampaignRequest{Name: " Autumn Twilight ", OwnerName: " louis "})
	createEnvelope = mustValidateTransportCommand(t, createEnvelope)
	createMessage, err := command.MessageAs[campaign.Create](createEnvelope)
	if err != nil {
		t.Fatalf("MessageAs(create) error = %v", err)
	}
	if got, want := createMessage.OwnerName, "louis"; got != want {
		t.Fatalf("owner name = %q, want %q", got, want)
	}

	createCharacterCmd, err := createCharacterEnvelope(&gamev1pb.CreateCharacterRequest{
		CampaignId:    "camp-1",
		ParticipantId: "part-1",
		Name:          " luna ",
	})
	if err != nil {
		t.Fatalf("createCharacterEnvelope() error = %v", err)
	}
	createCharacterCmd = mustValidateTransportCommand(t, createCharacterCmd)
	createCharacterMessage, err := command.MessageAs[character.Create](createCharacterCmd)
	if err != nil {
		t.Fatalf("MessageAs(create character) error = %v", err)
	}
	if got, want := createCharacterMessage.Name, "luna"; got != want {
		t.Fatalf("character name = %q, want %q", got, want)
	}

	joinEnvelope, err := createParticipantEnvelope(&gamev1pb.CreateParticipantRequest{
		CampaignId: "camp-1",
		Name:       " zoe ",
		Access:     gamev1pb.ParticipantAccess_PARTICIPANT_ACCESS_MEMBER,
	})
	if err != nil {
		t.Fatalf("createParticipantEnvelope() error = %v", err)
	}
	joinEnvelope = mustValidateTransportCommand(t, joinEnvelope)
	joinMessage, err := command.MessageAs[participant.Join](joinEnvelope)
	if err != nil {
		t.Fatalf("MessageAs(create participant) error = %v", err)
	}
	if got, want := joinMessage.Access, participant.AccessMember; got != want {
		t.Fatalf("access = %q, want %q", got, want)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(subjectIDHeader, "subject-1"))
	act, err := callerFromContext(ctx)
	if err != nil {
		t.Fatalf("callerFromContext() error = %v", err)
	}
	if got, want := act.SubjectID, "subject-1"; got != want {
		t.Fatalf("subject id = %q, want %q", got, want)
	}
	if _, err := callerFromContext(context.Background()); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("callerFromContext(no metadata) code = %v, want %v", status.Code(err), codes.Unauthenticated)
	}
}

func TestDomainEnumsRejectUnspecifiedValues(t *testing.T) {
	t.Parallel()

	if _, err := domainAccess(gamev1pb.ParticipantAccess_PARTICIPANT_ACCESS_UNSPECIFIED); err == nil {
		t.Fatal("domainAccess(unspecified) error = nil, want failure")
	}
}
