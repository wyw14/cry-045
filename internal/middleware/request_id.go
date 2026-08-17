package middleware

import (
	"context"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "request_id"
const ActorIDKey = "actor_id"

type requestMetadata struct {
	RequestID string
	ActorID   string
	Redacted  bool
}

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{3,96}$`)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" || !requestIDPattern.MatchString(id) {
			id = uuid.NewString()
		}
		actor := strings.TrimSpace(c.GetHeader("X-Actor-ID"))
		c.Set(RequestIDKey, id)
		c.Set(ActorIDKey, actor)
		c.Set("request_meta", requestMetadata{RequestID: id, ActorID: actor, Redacted: true})
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func ContextWithRequestMetadata(ctx context.Context, requestID, actorID string) context.Context {
	return context.WithValue(context.WithValue(ctx, requestContextKey("request_id"), requestID), requestContextKey("actor_id"), actorID)
}

func RequestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestContextKey("request_id")).(string)
	return value
}
func ActorIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestContextKey("actor_id")).(string)
	return value
}

type requestContextKey string
