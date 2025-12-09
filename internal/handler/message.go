package handler

import (
	"context"
	"fmt"
	"strings"

	"whatsappbot/internal/mikrotik"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

type MessageHandler struct {
	Client   *whatsmeow.Client
	Mikrotik *mikrotik.Manager
}

func NewMessageHandler(client *whatsmeow.Client, mk *mikrotik.Manager) *MessageHandler {
	return &MessageHandler{
		Client:   client,
		Mikrotik: mk,
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

	rawText := text
	text = strings.ToLower(strings.TrimSpace(text))

	if strings.HasPrefix(text, "carinama ") {
		h.reply(evt, "⏳ Mohon menunggu. Server sedang menangani permintaan anda!")

		namaUser := strings.TrimSpace(rawText[9:])

		go func() {
			hasil := h.Mikrotik.CariPengguna(namaUser)
			h.reply(evt, hasil)
		}()
		return
	}

	if strings.HasPrefix(text, "cariip ") {
		h.reply(evt, "⏳ Mohon menunggu. Server sedang menangani permintaan anda!")

		alamatIP := strings.TrimSpace(text[7:])

		go func() {
			hasil := h.Mikrotik.CariAlamatIP(alamatIP)
			h.reply(evt, hasil)
		}()
		return
	}

	if strings.HasPrefix(text, "hidupkan ") {
		h.reply(evt, "⏳ Mohon menunggu. Server sedang menangani permintaan anda!")

		ipUser := strings.TrimSpace(text[9:])
		if isValidIP(ipUser) {
			go func() {
				hasil := h.Mikrotik.UbahStatusIP(ipUser, "hidupkan")
				h.reply(evt, hasil)
			}()
		} else {
			h.reply(evt, "❌ Mohon Maaf ip yang anda masukkan kurang lengkap atau salah.")
		}
		return
	}

	if strings.HasPrefix(text, "matikan ") {
		h.reply(evt, "⏳ Mohon menunggu. Server sedang menangani permintaan anda!")

		ipUser := strings.TrimSpace(text[8:])
		if isValidIP(ipUser) {
			go func() {
				hasil := h.Mikrotik.UbahStatusIP(ipUser, "matikan")
				h.reply(evt, hasil)
			}()
		} else {
			h.reply(evt, "❌ Mohon Maaf ip yang anda masukkan kurang lengkap atau salah.")
		}
		return
	}

	if strings.HasPrefix(text, "tambahclient ") {
		h.reply(evt, "⏳ Mohon menunggu. Server sedang menangani permintaan anda!")
		args := strings.Fields(rawText)

		if len(args) >= 4 {
			alamatIP := args[1]
			queue := args[2]
			nama := strings.Join(args[3:], " ")

			if isValidIP(alamatIP) {
				go func() {
					hasil := h.Mikrotik.EditAtauTambahClient(alamatIP, queue, nama)
					h.reply(evt, hasil)
				}()
			} else {
				h.reply(evt, "❌ Alamat IP Salah atau kurang!")
			}
		} else {
			h.reply(evt, "❌ Format salah. Contoh: tambahclient 192.168.1.10 5M/5M UserBaru")
		}
		return
	}

	if strings.HasPrefix(text, "editlimit ") {
		h.reply(evt, "⏳ Mohon menunggu. Server sedang menangani permintaan anda!")

		args := strings.Fields(text)
		if len(args) >= 3 {
			alamatIP := args[1]
			queue := args[2]
			if isValidIP(alamatIP) && strings.Contains(queue, "/") {
				go func() {
					hasil := h.Mikrotik.EditLimit(alamatIP, queue)
					h.reply(evt, hasil)
				}()
			} else {
				h.reply(evt, "❌ Data Tidak Lengkap atau Format IP/Limit salah.")
			}
		} else {
			h.reply(evt, "❌ Mohon Maaf Data Tidak Lengkap. Format: editlimit <ip> <limit>")
		}
		return
	}

	if strings.HasPrefix(text, "tambahlimit ") {
		h.reply(evt, "⏳ Mohon menunggu. Server sedang menangani permintaan anda!")

		args := strings.Fields(rawText)
		if len(args) >= 4 {
			alamatIP := args[1]
			queue := args[2]
			nama := strings.Join(args[3:], " ")

			if isValidIP(alamatIP) && strings.Contains(queue, "/") {
				go func() {
					hasil := h.Mikrotik.EditAtauTambahClient(alamatIP, queue, nama)
					h.reply(evt, hasil)
				}()
			} else {
				h.reply(evt, "❌ Data Tidak Lengkap atau Format IP/Limit salah.")
			}
		} else {
			h.reply(evt, "❌ Mohon Maaf Data Tidak Lengkap.")
		}
		return
	}

	if text == "bantuan" || text == "menu" || text == "help" {
		menu := `🤖 *MENU BOT MIKROTIK*

*Cari berdasarkan nama*
Command: carinama <namaclient>
Contoh: carinama daffa

*Cari berdasarkan IP*
Command: cariip <ipclient>
Contoh: cariip 192.168.10.121

*Hidupkan pelanggan*
Command: hidupkan <ipclient>
Contoh: hidupkan 192.168.10.121

*Disable pelanggan*
Command: matikan <ipclient>
Contoh: matikan 192.168.10.121

*Tambah pelanggan*
Command: tambahclient <ip> <limit> <nama>
Contoh: tambahclient 192.168.10.122 10/10 userbaru

*Edit bandwidth client*
Command: editlimit <ip> <limit>
Contoh: editlimit 192.168.7.250 10/10

*Tambah bandwidth client*
Command: tambahlimit <ip> <limit> <nama>
Contoh: tambahlimit 192.168.7.250 10/10 nabil`

		h.reply(evt, menu)
		return
	}
}

func (h *MessageHandler) reply(evt *events.Message, responseText string) {
	_, err := h.Client.SendMessage(context.Background(), evt.Info.Chat, &waE2E.Message{
		Conversation: &responseText,
	})
	if err != nil {
		fmt.Printf("Error mengirim pesan: %v\n", err)
	}
}

func isValidIP(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	return strings.HasPrefix(ip, "192.168") ||
		strings.HasPrefix(ip, "123.123") ||
		strings.HasPrefix(ip, "172.16") ||
		strings.HasPrefix(ip, "193.168")
}
