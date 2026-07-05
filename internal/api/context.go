package api

import (
	"context"

	"goq/db/generated"
)

type contextKey int

const topicKey contextKey = iota

func withTopic(ctx context.Context, topic generated.Topic) context.Context {
	return context.WithValue(ctx, topicKey, topic)
}

func topicFromCtx(ctx context.Context) generated.Topic {
	return ctx.Value(topicKey).(generated.Topic)
}
