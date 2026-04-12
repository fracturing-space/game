package gamev1

import gamev1 "github.com/fracturing-space/game/api/gen/go/game/v1"

// StreamCampaignEvents replays persisted events after one sequence and then
// tails the campaign until the caller cancels. Read authorization is evaluated
// once when the stream opens and is not re-checked mid-stream.
func (s *Server) StreamCampaignEvents(req *gamev1.StreamCampaignEventsRequest, stream gamev1.GameService_StreamCampaignEventsServer) error {
	svc, err := s.requireService()
	if err != nil {
		return err
	}
	if err := requireStream(stream); err != nil {
		return err
	}
	if err := requireRequest(req); err != nil {
		return err
	}
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return invalidArgument(err)
	}
	act, err := callerFromContext(stream.Context())
	if err != nil {
		return err
	}

	events, err := svc.SubscribeEvents(stream.Context(), act, campaignID, req.GetAfterSeq())
	if err != nil {
		return mapDomainError(err)
	}
	defer events.Close()

	for record := range events.Records {
		protoEvent, err := protoStoredEvent(record)
		if err != nil {
			return internalStatus(err)
		}
		if err := stream.Send(protoEvent); err != nil {
			return err
		}
	}
	return nil
}
