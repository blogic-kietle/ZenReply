// Package handler provides HTTP request handlers for the ZenReply API.
package handler

import (
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/kietle/zenreply/pkg/response"
	"github.com/slack-go/slack/slackevents"
)

// SlackEventsWebhook godoc
//
//	@Summary		Slack Events API webhook
//	@Description	Receives DM events from Slack and triggers auto-replies using the recipient's own User Token (xoxp-). Handles URL verification challenge automatically.
//	@Tags			slack
//	@Accept			json
//	@Produce		json
//	@Param			X-Slack-Signature			header	string	true	"Slack request signature"
//	@Param			X-Slack-Request-Timestamp	header	string	true	"Slack request timestamp"
//	@Success		200
//	@Failure		400	{object}	response.Response
//	@Router			/slack/events [post]
func (h *Handler) SlackEventsWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "READ_ERROR", "failed to read request body", "")
		return
	}

	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		response.BadRequest(c, "PARSE_ERROR", "failed to parse slack event", err.Error())
		return
	}

	// Respond to Slack's URL verification challenge.
	if eventsAPIEvent.Type == slackevents.URLVerification {
		var challengeReq slackevents.EventsAPIURLVerificationEvent
		if err := json.Unmarshal(body, &challengeReq); err != nil {
			response.BadRequest(c, "PARSE_ERROR", "failed to parse challenge", "")
			return
		}
		c.JSON(200, gin.H{"challenge": challengeReq.Challenge})
		return
	}

	// Process callback events asynchronously so Slack doesn't time out.
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		// Parse the outer callback event to access AuthedUsers.
		var callbackEvent slackevents.EventsAPICallbackEvent
		if err := json.Unmarshal(body, &callbackEvent); err != nil {
			c.Status(200)
			return
		}

		go func() {
			switch ev := eventsAPIEvent.InnerEvent.Data.(type) {
			case *slackevents.MessageEvent:
				// Ignore bot messages and message edits/deletes.
				if ev.BotID != "" || ev.SubType != "" {
					return
				}
				// Only handle Direct Messages (im) and Multi-party DMs (mpim).
				if ev.ChannelType != "im" && ev.ChannelType != "mpim" {
					return
				}

				// Slack populates ev.User as the SENDER.
				// The RECEIVER (ZenReply user) is in callbackEvent.AuthedUsers.
				ownerSlackID := getOwnerFromAuthedUsers(callbackEvent.AuthedUsers, ev.User)
				if ownerSlackID == "" {
					return // Cannot determine owner.
				}

				_ = h.deepWorkService.HandleIncomingMessage(
					ctx,
					ownerSlackID,
					ev.User,
					ev.Channel,
					ev.Text,
					ev.TimeStamp,
					ev.ThreadTimeStamp,
				)
			}
		}()
	}

	c.Status(200)
}

// getOwnerFromAuthedUsers extracts the ZenReply user's Slack ID from the authed_users list.
// Slack includes the list of users who authorized the app in the outer callback event.
// We pick the first authed user that is NOT the sender.
func getOwnerFromAuthedUsers(authedUsers []string, senderID string) string {
	for _, uid := range authedUsers {
		if uid != senderID {
			return uid
		}
	}
	return ""
}
