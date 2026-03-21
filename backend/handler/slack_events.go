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
//	@Description	Receives and processes events from the Slack Events API (webhook mode). Handles URL verification and message events.
//	@Tags			slack
//	@Accept			json
//	@Produce		json
//	@Param			X-Slack-Signature			header	string	true	"Slack request signature"
//	@Param			X-Slack-Request-Timestamp	header	string	true	"Slack request timestamp"
//	@Success		200
//	@Failure		400	{object}	response.Response
//	@Failure		401	{object}	response.Response
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

	// Handle Slack URL verification challenge.
	if eventsAPIEvent.Type == slackevents.URLVerification {
		var challengeReq slackevents.EventsAPIURLVerificationEvent
		if err := json.Unmarshal(body, &challengeReq); err != nil {
			response.BadRequest(c, "PARSE_ERROR", "failed to parse challenge", "")
			return
		}
		c.JSON(200, gin.H{"challenge": challengeReq.Challenge})
		return
	}

	// Process callback events asynchronously.
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		go func() {
			switch ev := eventsAPIEvent.InnerEvent.Data.(type) {
			case *slackevents.MessageEvent:
				if ev.BotID != "" || ev.SubType == "bot_message" {
					return
				}
				if ev.ChannelType == "im" || ev.ChannelType == "mpim" {
					// In webhook mode, the bot user ID is the "owner" receiving the DM.
					// The workspace bot token is used to identify the owner.
					// For simplicity, we pass the bot user as the owner identifier.
					_ = h.deepWorkService.HandleIncomingMessage(
						ctx,
						ev.User, // In DM context, this is the sender; owner lookup is done inside the service.
						ev.User,
						ev.Channel,
						ev.Text,
						ev.TimeStamp,
						ev.ThreadTimeStamp,
					)
				}
			}
		}()
	}

	c.Status(200)
}
