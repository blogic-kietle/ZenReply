// Package slack provides Slack API integration utilities for ZenReply.
package slack

import (
	"context"
	"log/slog"

	slacklib "github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// EventHandler is a callback function invoked when a relevant Slack message event is received.
type EventHandler func(ctx context.Context, ev interface{})

// SocketModeClient wraps the Slack Socket Mode client and dispatches events.
type SocketModeClient struct {
	client  *socketmode.Client
	handler EventHandler
	logger  *slog.Logger
}

// NewSocketModeClient creates a new SocketModeClient.
// appToken must be an App-Level Token (xapp-...) and botToken a Bot Token (xoxb-...).
func NewSocketModeClient(appToken, botToken string, handler EventHandler, logger *slog.Logger) *SocketModeClient {
	api := slacklib.New(
		botToken,
		slacklib.OptionAppLevelToken(appToken),
	)
	client := socketmode.New(api,
		socketmode.OptionDebug(false),
	)
	return &SocketModeClient{
		client:  client,
		handler: handler,
		logger:  logger,
	}
}

// Run starts the Socket Mode event loop. It blocks until the context is cancelled.
func (s *SocketModeClient) Run(ctx context.Context) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-s.client.Events:
				if !ok {
					return
				}
				s.dispatch(ctx, evt)
			}
		}
	}()

	return s.client.RunContext(ctx)
}

func (s *SocketModeClient) dispatch(ctx context.Context, evt socketmode.Event) {
	switch evt.Type {
	case socketmode.EventTypeEventsAPI:
		eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
		if !ok {
			return
		}
		s.client.Ack(*evt.Request)

		if eventsAPIEvent.Type == slackevents.CallbackEvent {
			s.handleCallbackEvent(ctx, eventsAPIEvent)
		}

	default:
		if evt.Request != nil {
			s.client.Ack(*evt.Request)
		}
	}
}

func (s *SocketModeClient) handleCallbackEvent(ctx context.Context, event slackevents.EventsAPIEvent) {
	switch ev := event.InnerEvent.Data.(type) {
	case *slackevents.MessageEvent:
		if ev.BotID != "" || ev.SubType == "bot_message" {
			return
		}
		if ev.ChannelType == "im" || ev.ChannelType == "mpim" {
			s.logger.Info("received DM event",
				slog.String("from", ev.User),
				slog.String("channel", ev.Channel),
			)
			if s.handler != nil {
				s.handler(ctx, ev)
			}
		}

	case *slackevents.AppMentionEvent:
		s.logger.Info("received app mention",
			slog.String("from", ev.User),
			slog.String("channel", ev.Channel),
		)
		msgEv := &slackevents.MessageEvent{
			Type:            ev.Type,
			User:            ev.User,
			Text:            ev.Text,
			TimeStamp:       ev.TimeStamp,
			Channel:         ev.Channel,
			ChannelType:     "channel",
			ThreadTimeStamp: ev.ThreadTimeStamp,
		}
		if s.handler != nil {
			s.handler(ctx, msgEv)
		}
	}
}
