package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"whatsappbot/internal/handler"
	"whatsappbot/internal/service"
)

func main() {
	fmt.Println("Memulai WhatsApp Bot...")

	client, err := service.InitWhatsApp()
	if err != nil {
		panic(err)
	}
	msgHandler := handler.NewMessageHandler(client)

	client.AddEventHandler(msgHandler.HandleEvent)

	fmt.Println("Bot berjalan! Tekan Ctrl+C untuk berhenti.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
	fmt.Println("Bot berhenti.")
}
