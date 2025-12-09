package mikrotik

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"

	"whatsappbot/internal/config"

	"github.com/go-routeros/routeros"
)

type Manager struct {
	Config *config.Config
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		Config: cfg,
	}
}

// --- HELPER KONEKSI (ON-THE-FLY) ---

// Fungsi ini membuat koneksi BARU setiap kali dipanggil, lalu mengembalikan clientnya.
// Jangan lupa di-close setelah dipakai!
func (m *Manager) connectToRouter(routerID int) (*routeros.Client, string, error) {
	var host, user, pass, name string

	if routerID == 1 {
		host = m.Config.Router1.Host
		user = m.Config.Router1.User
		pass = m.Config.Router1.Pass
		name = "Kampoeng IT"
	} else {
		host = m.Config.Router2.Host
		user = m.Config.Router2.User
		pass = m.Config.Router2.Pass
		name = "Kampoeng Putih"
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	// DialTLS ke Router
	client, err := routeros.DialTLS(host, user, pass, tlsConfig)
	if err != nil {
		return nil, name, fmt.Errorf("gagal konek ke %s: %v", name, err)
	}

	return client, name, nil
}

// Menentukan ID Router berdasarkan IP
func (m *Manager) determineRouterID(ip string) (int, error) {
	if strings.HasPrefix(ip, "193.168") {
		return 2, nil // Router 2
	}
	if strings.HasPrefix(ip, "192.168") || strings.HasPrefix(ip, "123.123") || strings.HasPrefix(ip, "172.16") {
		return 1, nil // Router 1
	}
	return 0, fmt.Errorf("IP %s tidak dikenali", ip)
}

// --- HELPER FORMATTING ---

func formatSpeed(limitStr string) string {
	if limitStr == "" {
		return "tidak ada"
	}
	parts := strings.Split(limitStr, "/")
	if len(parts) != 2 {
		return limitStr
	}
	convert := func(valStr string) string {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return "N/A"
		}
		if val >= 1000000 {
			return fmt.Sprintf("%d Mbps", val/1000000)
		}
		if val >= 1000 {
			return fmt.Sprintf("%d Kbps", val/1000)
		}
		return fmt.Sprintf("%d bps", val)
	}
	return fmt.Sprintf("%s / %s", convert(parts[0]), convert(parts[1]))
}

func formatLimitToBytes(queue string) string {
	parts := strings.Split(queue, "/")
	if len(parts) != 2 {
		return queue
	}
	return fmt.Sprintf("%s000000/%s000000", parts[0], parts[1])
}

// --- FUNGSI UTAMA ---

func (m *Manager) GetIdentity(routerID int) string {
	// 1. Konek
	client, _, err := m.connectToRouter(routerID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	// 2. Wajib Close saat selesai
	defer client.Close()

	// 3. Eksekusi
	reply, err := client.Run("/system/identity/print")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if len(reply.Re) > 0 {
		return reply.Re[0].Map["name"]
	}
	return "Unknown"
}

// 1. CARI PENGGUNA (Login Router 1 -> Cek -> Close -> Login Router 2 -> Cek -> Close)
func (m *Manager) CariPengguna(nama string) string {
	var results []string
	routerIDs := []int{1, 2}

	for _, rid := range routerIDs {
		// Konek
		client, rName, err := m.connectToRouter(rid)
		if err != nil {
			continue // Skip router ini jika mati
		}

		// Ambil data Binding
		reply, err := client.Run("/ip/hotspot/ip-binding/print")
		if err == nil {
			for _, re := range reply.Re {
				comment := re.Map["comment"]
				// Filter manual logic
				if strings.Contains(strings.ToLower(comment), strings.ToLower(nama)) {
					address := re.Map["address"]
					status := "Aktif"
					if re.Map["disabled"] == "true" {
						status = "Isolir"
					}

					// Cek Queue (Masih dalam satu koneksi yang sama)
					limitStr := "tidak ada"
					qReply, errQ := client.Run("/queue/simple/print", "?target="+address+"/32")
					if errQ == nil && len(qReply.Re) > 0 {
						limitStr = formatSpeed(qReply.Re[0].Map["max-limit"])
					}

					msg := fmt.Sprintf("*Lokasi*: %s\n*comment*: %s\n*address*: %s\n*status*: %s\n*max limit*: %s\n",
						rName, comment, address, status, limitStr)
					results = append(results, msg)
				}
			}
		}

		// Tutup koneksi router ini sebelum lanjut ke router berikutnya
		client.Close()
	}

	if len(results) == 0 {
		return fmt.Sprintf("Tidak ditemukan pengguna dengan nama '%s'", nama)
	}
	return strings.Join(results, "\n")
}

// 2. CARI ALAMAT IP
func (m *Manager) CariAlamatIP(ip string) string {
	routerIDs := []int{1, 2}

	for _, rid := range routerIDs {
		client, rName, err := m.connectToRouter(rid)
		if err != nil {
			continue
		}

		// Cek Binding
		reply, err := client.Run("/ip/hotspot/ip-binding/print", "?address="+ip)

		// Jika ketemu
		if err == nil && len(reply.Re) > 0 {
			item := reply.Re[0].Map
			status := "Aktif"
			if item["disabled"] == "true" {
				status = "Isolir"
			}

			// Cek Queue
			limitStr := "tidak ada"
			qReply, errQ := client.Run("/queue/simple/print", "?target="+ip+"/32")
			if errQ == nil && len(qReply.Re) > 0 {
				limitStr = formatSpeed(qReply.Re[0].Map["max-limit"])
			}

			client.Close() // Tutup
			return fmt.Sprintf("*Sumber*: %s\n*comment*: %s\n*address*: %s\n*status*: %s\n*max limit*: %s",
				rName, item["comment"], ip, status, limitStr)
		}

		client.Close() // Tutup dan lanjut router sebelah
	}

	return fmt.Sprintf("IP %s tidak ditemukan di Router 1 maupun Router 2.", ip)
}

// 3. UBAH STATUS
func (m *Manager) UbahStatusIP(ip string, command string) string {
	rid, err := m.determineRouterID(ip)
	if err != nil {
		return err.Error()
	}

	// Konek Spesifik
	client, rName, err := m.connectToRouter(rid)
	if err != nil {
		return err.Error()
	}
	defer client.Close()

	// Cari ID
	reply, err := client.Run("/ip/hotspot/ip-binding/print", "?address="+ip)
	if err != nil || len(reply.Re) == 0 {
		return fmt.Sprintf("IP %s tidak ditemukan di %s.", ip, rName)
	}

	id := reply.Re[0].Map[".id"]
	comment := reply.Re[0].Map["comment"]
	currDisabled := reply.Re[0].Map["disabled"]

	target := "no"
	if command == "matikan" {
		target = "yes"
	}

	if command == "matikan" && currDisabled == "true" {
		return fmt.Sprintf("*Lokasi*: %s\n*User*: %s\n*Status*: Sudah Isolir", rName, comment)
	}
	if command == "hidupkan" && currDisabled == "false" {
		return fmt.Sprintf("*Lokasi*: %s\n*User*: %s\n*Status*: Sudah Hidup", rName, comment)
	}

	// Eksekusi
	_, err = client.Run("/ip/hotspot/ip-binding/set", "=.id="+id, "=disabled="+target)
	if err != nil {
		return fmt.Sprintf("Gagal ubah status: %v", err)
	}

	msg := "berhasil di-hidupkan"
	if command == "matikan" {
		msg = "berhasil di-disable"
	}
	return fmt.Sprintf("*Lokasi*: %s\n*User*: %s\n*Status*: %s", rName, comment, msg)
}

// 4. TAMBAH / EDIT CLIENT
func (m *Manager) EditAtauTambahClient(ip, limit, nama string) string {
	rid, err := m.determineRouterID(ip)
	if err != nil {
		return err.Error()
	}

	limitBytes := formatLimitToBytes(limit)

	// Konek
	client, rName, err := m.connectToRouter(rid)
	if err != nil {
		return err.Error()
	}
	// Kita jangan defer Close disini, karena di akhir kita panggil CariAlamatIP (yang akan connect lagi)
	// Jadi kita close manual sebelum return.

	// A. Binding
	reply, _ := client.Run("/ip/hotspot/ip-binding/print", "?address="+ip)
	if len(reply.Re) == 0 {
		_, err = client.Run("/ip/hotspot/ip-binding/add", "=address="+ip, "=type=bypassed", "=disabled=no", "=comment="+nama)
	} else {
		id := reply.Re[0].Map[".id"]
		_, err = client.Run("/ip/hotspot/ip-binding/set", "=.id="+id, "=comment="+nama)
	}
	if err != nil {
		client.Close()
		return fmt.Sprintf("Gagal proses binding di %s: %v", rName, err)
	}

	// B. Queue
	qReply, _ := client.Run("/queue/simple/print", "?target="+ip+"/32")
	if len(qReply.Re) == 0 {
		_, err = client.Run("/queue/simple/add", "=name="+nama, "=target="+ip+"/32", "=max-limit="+limitBytes, "=comment="+nama+" - nd")
	} else {
		id := qReply.Re[0].Map[".id"]
		_, err = client.Run("/queue/simple/set", "=.id="+id, "=name="+nama, "=max-limit="+limitBytes, "=comment="+nama+" - nd")
	}

	if err != nil {
		client.Close()
		return fmt.Sprintf("Binding sukses, tapi Queue gagal: %v", err)
	}

	// Tutup koneksi SEBELUM memanggil CariAlamatIP
	client.Close()

	// Panggil fungsi CariAlamatIP (Fungsi ini akan membuka koneksi baru lagi)
	return m.CariAlamatIP(ip)
}

// 5. PUTUS CLIENT
func (m *Manager) PutusClient(ip string) string {
	rid, err := m.determineRouterID(ip)
	if err != nil {
		return err.Error()
	}

	client, rName, err := m.connectToRouter(rid)
	if err != nil {
		return err.Error()
	}
	defer client.Close()

	// Hapus Binding
	reply, _ := client.Run("/ip/hotspot/ip-binding/print", "?address="+ip)
	if len(reply.Re) > 0 {
		id := reply.Re[0].Map[".id"]
		client.Run("/ip/hotspot/ip-binding/remove", "=.id="+id)
	} else {
		return fmt.Sprintf("IP %s tidak ada di binding %s", ip, rName)
	}

	// Hapus Queue
	qReply, _ := client.Run("/queue/simple/print", "?target="+ip+"/32")
	if len(qReply.Re) > 0 {
		id := qReply.Re[0].Map[".id"]
		client.Run("/queue/simple/remove", "=.id="+id)
	}

	return fmt.Sprintf("Client %s berhasil diputus dari %s", ip, rName)
}

// 6. EDIT LIMIT
func (m *Manager) EditLimit(ip, limit string) string {
	rid, err := m.determineRouterID(ip)
	if err != nil {
		return err.Error()
	}

	limitBytes := formatLimitToBytes(limit)

	client, rName, err := m.connectToRouter(rid)
	if err != nil {
		return err.Error()
	}

	// Cek Queue
	qReply, err := client.Run("/queue/simple/print", "?target="+ip+"/32")
	if err != nil || len(qReply.Re) == 0 {
		client.Close()
		return fmt.Sprintf("IP %s belum dilimit di %s", ip, rName)
	}

	id := qReply.Re[0].Map[".id"]
	name := qReply.Re[0].Map["name"]

	_, err = client.Run("/queue/simple/set", "=.id="+id, "=name="+name, "=max-limit="+limitBytes)
	if err != nil {
		client.Close()
		return fmt.Sprintf("Gagal edit limit: %v", err)
	}

	client.Close() // Tutup sebelum panggil fungsi lain
	return m.CariAlamatIP(ip)
}
