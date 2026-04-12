package eventjournal

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"

	"github.com/fracturing-space/game/internal/event"
)

type eventEnvelopeCodec struct {
	catalog *event.Catalog
}

func newEventEnvelopeCodec(catalog *event.Catalog) (*eventEnvelopeCodec, error) {
	if catalog == nil {
		return nil, fmt.Errorf("event catalog is required")
	}
	return &eventEnvelopeCodec{catalog: catalog}, nil
}

func (c *eventEnvelopeCodec) Encode(envelope event.Envelope) (string, []byte, error) {
	validated, _, err := c.catalog.Validate(envelope)
	if err != nil {
		return "", nil, err
	}
	var payload bytes.Buffer
	if err := gob.NewEncoder(&payload).Encode(validated.Message); err != nil {
		return "", nil, err
	}
	return string(validated.Type()), payload.Bytes(), nil
}

func (c *eventEnvelopeCodec) Decode(campaignID string, eventType event.Type, payloadBlob []byte) (event.Envelope, error) {
	spec, ok := c.catalog.SpecFor(eventType)
	if !ok {
		return event.Envelope{}, fmt.Errorf("event type is not registered: %s", eventType)
	}
	message, err := decodeEventMessage(spec.Definition().MessageType, payloadBlob)
	if err != nil {
		return event.Envelope{}, err
	}
	validated, _, err := c.catalog.Validate(event.Envelope{
		CampaignID: campaignID,
		Message:    message,
	})
	if err != nil {
		return event.Envelope{}, err
	}
	return validated, nil
}

func decodeEventMessage(messageType reflect.Type, payloadBlob []byte) (event.Message, error) {
	if messageType == nil {
		return nil, fmt.Errorf("event message type is required")
	}
	target := reflect.New(messageType)
	if messageType.Kind() == reflect.Pointer {
		target = reflect.New(messageType.Elem())
	}
	if err := gob.NewDecoder(bytes.NewReader(payloadBlob)).Decode(target.Interface()); err != nil {
		return nil, err
	}
	if messageType.Kind() == reflect.Pointer {
		message, ok := target.Interface().(event.Message)
		if !ok {
			return nil, fmt.Errorf("decoded event message does not implement event.Message: %v", messageType)
		}
		return message, nil
	}
	message, ok := target.Elem().Interface().(event.Message)
	if !ok {
		return nil, fmt.Errorf("decoded event message does not implement event.Message: %v", messageType)
	}
	return message, nil
}
