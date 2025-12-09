package handler

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

type MessageHandler struct {
	Client *whatsmeow.Client
}

func NewMessageHandler(client *whatsmeow.Client) *MessageHandler {
	return &MessageHandler{
		Client: client,
	}
}

func (h *MessageHandler) HandleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		h.processMessage(v)
	}
}

func (h *MessageHandler) processMessage(evt *events.Message) {
	if evt.Info.IsFromMe {
		return
	}
	text := ""
	if evt.Message.Conversation != nil {
		text = *evt.Message.Conversation
	} else if evt.Message.ExtendedTextMessage != nil {
		text = *evt.Message.ExtendedTextMessage.Text
	}

	text = strings.ToLower(strings.TrimSpace(text))
	if text == "ping" {
		fmt.Println("Menerima ping dari:", evt.Info.Sender.User)
		h.reply(evt, "Pong! 🏓 Bot WhatsApp Go siap melayani.")
	}
}

func (h *MessageHandler) reply(evt *events.Message, responseText string) {
	_, err := h.Client.SendMessage(context.Background(), evt.Info.Chat, &waE2E.Message{
		Conversation: &responseText,
	})
	if err != nil {
		fmt.Printf("Gagal mengirim pesan: %v\n", err)
	}
}
