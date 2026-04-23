package repo

import (
	"context"
	"encoding/json"
	"eraya/chat"
	"eraya/domain"
	"log"

	"github.com/redis/go-redis/v9"
)

type chatPubSub struct {
	rdb *redis.Client
}

func NewChatPubSub(rdb *redis.Client) chat.ChatPubSub {
	return &chatPubSub{rdb: rdb}
}

func (p *chatPubSub) PublishMessage(channel string, msg *domain.Message) error {
	data, _ := json.Marshal(msg)
	return p.rdb.Publish(context.Background(), channel, data).Err()
}

func (p *chatPubSub) SubscribeToMessages(channel string, handler func(msg *domain.Message)) error {
	sub := p.rdb.Subscribe(context.Background(), channel)
	ch := sub.Channel()

	go func() {
		for msg := range ch {
			var m domain.Message
			if err := json.Unmarshal([]byte(msg.Payload), &m); err == nil {
				handler(&m)
			} else {
				log.Printf("Failed to unmarshal chat message: %v", err)
			}
		}
	}()
	return nil
}
