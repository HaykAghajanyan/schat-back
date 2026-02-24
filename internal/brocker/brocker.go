package broker

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
)

const messageChannel = "chat:messages"

type Message struct {
	UserID  int    `json:"user_id"`
	Payload []byte `json:"payload"`
}

type Broker struct {
	client *redis.Client
}

func New(addr string) *Broker {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("broker: failed to connect to Redis at %s: %v", addr, err)
	}

	return &Broker{client: client}
}

// Publish sends a message to the Redis channel
func (b *Broker) Publish(ctx context.Context, userID int, payload []byte) error {
	msg := Message{UserID: userID, Payload: payload}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return b.client.Publish(ctx, messageChannel, data).Err()
}

// Subscribe listens for messages and calls handler for each one
func (b *Broker) Subscribe(ctx context.Context, handler func(userID int, payload []byte)) {
	sub := b.client.Subscribe(ctx, messageChannel)
	ch := sub.Channel()

	go func() {
		for msg := range ch {
			var m Message
			if err := json.Unmarshal([]byte(msg.Payload), &m); err != nil {
				continue
			}
			handler(m.UserID, m.Payload)
		}
	}()
}
