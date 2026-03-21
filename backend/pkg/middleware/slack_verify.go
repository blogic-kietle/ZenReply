// Package middleware provides HTTP middleware for the ZenReply API.
package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kietle/zenreply/pkg/response"
)

const (
	slackSignatureHeader  = "X-Slack-Signature"
	slackTimestampHeader  = "X-Slack-Request-Timestamp"
	slackSignatureVersion = "v0"
	// maxTimestampAge is the maximum allowed age of a Slack request timestamp (5 minutes).
	maxTimestampAge = 5 * time.Minute
)

// SlackVerify returns a Gin middleware that verifies incoming Slack request signatures.
// It prevents replay attacks by rejecting requests older than 5 minutes.
func SlackVerify(signingSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if signingSecret == "" {
			c.Next()
			return
		}

		timestamp := c.GetHeader(slackTimestampHeader)
		if timestamp == "" {
			response.Unauthorized(c, "missing slack request timestamp")
			c.Abort()
			return
		}

		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			response.Unauthorized(c, "invalid slack request timestamp")
			c.Abort()
			return
		}

		// Reject requests older than maxTimestampAge to prevent replay attacks.
		if time.Since(time.Unix(ts, 0)) > maxTimestampAge {
			response.Unauthorized(c, "slack request timestamp is too old")
			c.Abort()
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.InternalServerError(c, "failed to read request body")
			c.Abort()
			return
		}
		// Restore body for subsequent handlers.
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		sigBaseString := fmt.Sprintf("%s:%s:%s", slackSignatureVersion, timestamp, string(body))
		mac := hmac.New(sha256.New, []byte(signingSecret))
		mac.Write([]byte(sigBaseString))
		expectedSig := slackSignatureVersion + "=" + hex.EncodeToString(mac.Sum(nil))

		receivedSig := c.GetHeader(slackSignatureHeader)
		if !hmac.Equal([]byte(expectedSig), []byte(receivedSig)) {
			response.Unauthorized(c, "invalid slack request signature")
			c.Abort()
			return
		}

		c.Next()
	}
}
