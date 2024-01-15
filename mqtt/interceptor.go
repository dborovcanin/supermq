package mqtt

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/mproxy/pkg/session"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"google.golang.org/protobuf/proto"
)

type interceptor struct{}

func NewInterceptor() session.Interceptor {
	return &interceptor{}
}

func (i *interceptor) Intercept(pkt packets.ControlPacket, dir session.Direction, ctx context.Context) (packets.ControlPacket, error) {
	if p, ok := pkt.(*packets.PublishPacket); ok {
		data := p.Payload
		s, ok := session.FromContext(ctx)
		if !ok {
			return nil, errors.Wrap(ErrFailedPublish, ErrClientNotInitialized)
		}
		switch dir {
		case session.Up:

			// h.logger.Info(fmt.Sprintf(LogInfoPublished, s.ID, *topic))
			// Topics are in the format:
			// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>

			channelParts := channelRegExp.FindStringSubmatch(p.TopicName)
			if len(channelParts) < 2 {
				return nil, errors.Wrap(ErrFailedPublish, ErrMalformedTopic)
			}

			chanID := channelParts[1]
			subtopic := channelParts[2]

			subtopic, err := parseSubtopic(subtopic)
			if err != nil {
				return nil, errors.Wrap(ErrFailedParseSubtopic, err)
			}

			msg := &messaging.Message{
				Protocol:  protocol,
				Channel:   chanID,
				Subtopic:  subtopic,
				Publisher: s.Username,
				Payload:   p.Payload,
				Created:   time.Now().UnixNano(),
			}
			data, err = proto.Marshal(msg)
			if err != nil {
				return nil, err
			}
		case session.Down:
			m := &messaging.Message{}
			if err := proto.Unmarshal(data, m); err != nil {
				return nil, err
			}
			data = m.Payload
		}
		p.Payload = data
	}
	return pkt, nil
}
