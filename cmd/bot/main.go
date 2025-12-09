package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"whatsappbot/internal/config"
	"whatsappbot/internal/handler"
	"whatsappbot/internal/mikrotik"
	"whatsappbot/internal/service"
)

func main() {
	fmt.Println("🚀 Memulai WhatsApp Bot System...")

	cfg := config.LoadConfig()
	fmt.Println("✅ Konfigurasi dimuat")
	mikrotikMgr := mikrotik.NewManager(cfg)

	client, err := service.InitWhatsApp()
	if err != nil {
		fmt.Printf("🔥 Fatal Error: %v\n", err)
		return
	}

	msgHandler := handler.NewMessageHandler(client, mikrotikMgr)

	client.AddEventHandler(msgHandler.HandleEvent)

	fmt.Println("✅ Bot Berjalan! Menunggu pesan...")
	fmt.Println("   (Tekan Ctrl+C untuk berhenti)")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c

	fmt.Println("\n⚠️  Menutup koneksi...")
	client.Disconnect()
	fmt.Println("👋 Bot Berhenti.")
}
