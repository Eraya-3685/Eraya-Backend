package repo

import (
	"context"
	"encoding/json"
	"eraya/chat"
	"eraya/domain"

	"github.com/redis/go-redis/v9"
)

type chatPubSub struct {
	rdb *redis.Client
}

func NewChatPubSub(rdb *redis.Client) chat.ChatPubSub {
	return &chatPubSub{rdb: rdb}
}

func (p *chatPubSub) PublishMessage(ctx context.Context, channel string, msg *domain.Message) error {
	data, _ := json.Marshal(msg)
	return p.rdb.Publish(ctx, channel, data).Err()
}

func (p *chatPubSub) SubscribeToMessages(ctx context.Context, channel string, handler func(msg *domain.Message)) error {
	sub := p.rdb.Subscribe(ctx, channel)
	ch := sub.Channel()

	go func() {
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var m domain.Message
				if err := json.Unmarshal([]byte(msg.Payload), &m); err == nil {
					handler(&m)
				}
			}
		}
	}()
	return nil
}
