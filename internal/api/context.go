package api

import (
	"context"

	"goq/db/store"
)

type contextKey int

const topicKey contextKey = iota

func withTopic(ctx context.Context, topic store.Topic) context.Context {
	return context.WithValue(ctx, topicKey, topic)
}

func topicFromCtx(ctx context.Context) store.Topic {
	return ctx.Value(topicKey).(store.Topic)
}
