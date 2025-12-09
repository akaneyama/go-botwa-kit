package service

import (
	"context"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func InitWhatsApp() (*whatsmeow.Client, error) {

	dbLog := waLog.Stdout("Database", "ERROR", true)
	container, err := sqlstore.New(context.Background(), "sqlite3", "file:store.db?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("gagal inisialisasi database: %v", err)
	}
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil device store: %v", err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			return nil, fmt.Errorf("gagal connect: %v", err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("Silakan scan QR Code di bawah ini:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			return nil, fmt.Errorf("gagal connect ulang: %v", err)
		}
	}

	return client, nil
}
