package mikrotik

import (
	"crypto/tls"
	"fmt"
	"strings"
	"sync"

	"whatsappbot/internal/config"

	"github.com/go-routeros/routeros"
)

type Manager struct {
	Client1 *routeros.Client
	Mu1     sync.Mutex // Kunci pengaman untuk Router 1

	Client2 *routeros.Client
	Mu2     sync.Mutex // Kunci pengaman untuk Router 2
}

// Helper: Koneksi SSL
func connectSSL(host, user, pass string) (*routeros.Client, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	// Pastikan host di config memiliki port, misal 192.168.88.1:8729
	return routeros.DialTLS(host, user, pass, tlsConfig)
}

func NewManager(cfg *config.Config) *Manager {
	mgr := &Manager{}

	c1, err := connectSSL(cfg.Router1.Host, cfg.Router1.User, cfg.Router1.Pass)
	if err != nil {
		fmt.Printf("⚠️ Gagal konek ke MikroTik 1 (SSL): %v\n", err)
	} else {
		mgr.Client1 = c1
		fmt.Println("✅ Terhubung ke MikroTik 1 via SSL")
	}

	c2, err := connectSSL(cfg.Router2.Host, cfg.Router2.User, cfg.Router2.Pass)
	if err != nil {
		fmt.Printf("⚠️ Gagal konek ke MikroTik 2 (SSL): %v\n", err)
	} else {
		mgr.Client2 = c2
		fmt.Println("✅ Terhubung ke MikroTik 2 via SSL")
	}

	return mgr
}

// --- HELPER FUNGSI ---

// Logika: 193.168 -> Router 2, Sisanya -> Router 1
// Mengembalikan: Client, Mutex, Nama Router, Error
func (m *Manager) getTargetRouter(ip string) (*routeros.Client, *sync.Mutex, string, error) {
	// Aturan Router 2 (Kampoeng Putih)
	if strings.HasPrefix(ip, "193.168") {
		if m.Client2 == nil {
			return nil, nil, "", fmt.Errorf("❌ Koneksi ke Router 2 (Kampoeng Putih) terputus")
		}
		return m.Client2, &m.Mu2, "Kampoeng Putih", nil
	}

	// Aturan Router 1 (Kampoeng IT) - Default
	if strings.HasPrefix(ip, "192.168") || strings.HasPrefix(ip, "123.123") || strings.HasPrefix(ip, "172.16") {
		if m.Client1 == nil {
			return nil, nil, "", fmt.Errorf("❌ Koneksi ke Router 1 (Kampoeng IT) terputus")
		}
		return m.Client1, &m.Mu1, "Kampoeng IT", nil
	}

	return nil, nil, "", fmt.Errorf("❌ IP %s tidak dikenali (cek prefix IP)", ip)
}

// --- FUNGSI UTAMA ---

func (m *Manager) GetIdentity(routerID int) string {
	var client *routeros.Client
	var mu *sync.Mutex

	if routerID == 1 {
		client = m.Client1
		mu = &m.Mu1
	} else {
		client = m.Client2
		mu = &m.Mu2
	}

	if client == nil {
		return "❌ Koneksi Putus"
	}

	mu.Lock()
	defer mu.Unlock()

	reply, err := client.Run("/system/identity/print")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if len(reply.Re) > 0 {
		return reply.Re[0].Map["name"]
	}
	return "Unknown"
}

// 1. CARI PENGGUNA (Search All Routers)
func (m *Manager) CariPengguna(nama string) string {
	var results []string

	targets := []struct {
		Client *routeros.Client
		Mu     *sync.Mutex
		Name   string
	}{
		{m.Client1, &m.Mu1, "Kampoeng IT"},
		{m.Client2, &m.Mu2, "Kampoeng Putih"},
	}

	for _, t := range targets {
		if t.Client == nil {
			continue
		}

		// KUNCI PINTU
		t.Mu.Lock()

		reply, err := t.Client.Run("/ip/hotspot/ip-binding/print")

		// Simpan data ke variabel lokal
		var foundItems []map[string]string
		if err == nil {
			for _, re := range reply.Re {
				if strings.Contains(strings.ToLower(re.Map["comment"]), strings.ToLower(nama)) {
					foundItems = append(foundItems, re.Map)
				}
			}
		}

		// Proses item yang ditemukan
		for _, item := range foundItems {
			comment := item["comment"]
			address := item["address"]
			disabled := item["disabled"]
			status := "Aktif"
			if disabled == "true" {
				status = "Isolir"
			}

			limitStr := "tidak ada"
			qReply, errQ := t.Client.Run("/queue/simple/print", "?target="+address+"/32")

			if errQ == nil && qReply != nil && len(qReply.Re) > 0 {
				rawLimit := qReply.Re[0].Map["max-limit"]
				limitStr = formatSpeed(rawLimit)
			}

			msg := fmt.Sprintf("*Lokasi*: %s\n*comment*: %s\n*address*: %s\n*status*: %s\n*max limit*: %s\n",
				t.Name, comment, address, status, limitStr)
			results = append(results, msg)
		}

		// BUKA KUNCI
		t.Mu.Unlock()
	}

	if len(results) == 0 {
		return fmt.Sprintf("Tidak ditemukan pengguna dengan nama '%s'", nama)
	}
	return strings.Join(results, "\n")
}

// 2. CARI ALAMAT IP (Search All Routers)
func (m *Manager) CariAlamatIP(ip string) string {
	targets := []struct {
		Client *routeros.Client
		Mu     *sync.Mutex
		Name   string
	}{
		{m.Client1, &m.Mu1, "Kampoeng IT"},
		{m.Client2, &m.Mu2, "Kampoeng Putih"},
	}

	for _, t := range targets {
		if t.Client == nil {
			continue
		}

		// KUNCI PINTU
		t.Mu.Lock()

		reply, err := t.Client.Run("/ip/hotspot/ip-binding/print", "?address="+ip)

		// Jika error atau kosong, buka kunci dan lanjut ke router sebelah
		if err != nil || reply == nil || len(reply.Re) == 0 {
			t.Mu.Unlock()
			continue
		}

		item := reply.Re[0].Map
		status := "Aktif"
		if item["disabled"] == "true" {
			status = "Isolir"
		}

		limitStr := "tidak ada"
		qReply, errQ := t.Client.Run("/queue/simple/print", "?target="+ip+"/32")

		if errQ == nil && qReply != nil && len(qReply.Re) > 0 {
			limitStr = formatSpeed(qReply.Re[0].Map["max-limit"])
		}

		// BUKA KUNCI
		t.Mu.Unlock()

		return fmt.Sprintf("*Sumber*: %s\n*comment*: %s\n*address*: %s\n*status*: %s\n*max limit*: %s",
			t.Name, item["comment"], ip, status, limitStr)
	}

	return fmt.Sprintf("IP %s tidak ditemukan di Router 1 maupun Router 2.", ip)
}

// 3. UBAH STATUS (HIDUPKAN/MATIKAN)
func (m *Manager) UbahStatusIP(ip string, command string) string {
	// 1. Tentukan Router dan Kunci
	client, mu, routerName, err := m.getTargetRouter(ip)
	if err != nil {
		return err.Error()
	}

	// KUNCI PINTU
	mu.Lock()
	defer mu.Unlock()

	// 2. Cari ID
	reply, err := client.Run("/ip/hotspot/ip-binding/print", "?address="+ip)
	if err != nil || reply == nil || len(reply.Re) == 0 {
		return fmt.Sprintf("IP %s tidak ditemukan di %s.", ip, routerName)
	}

	id := reply.Re[0].Map[".id"]
	comment := reply.Re[0].Map["comment"]
	currentDisabled := reply.Re[0].Map["disabled"]

	// 3. Logika Matikan/Hidupkan
	targetDisabled := "no"
	if command == "matikan" {
		targetDisabled = "yes"
	}

	if command == "matikan" && currentDisabled == "true" {
		return fmt.Sprintf("*Lokasi*: %s\n*User*: %s\n*Status*: Sudah Isolir", routerName, comment)
	}
	if command == "hidupkan" && currentDisabled == "false" {
		return fmt.Sprintf("*Lokasi*: %s\n*User*: %s\n*Status*: Sudah Hidup", routerName, comment)
	}

	// 4. Eksekusi
	_, err = client.Run("/ip/hotspot/ip-binding/set", "=.id="+id, "=disabled="+targetDisabled)
	if err != nil {
		return fmt.Sprintf("Gagal mengubah status di %s: %v", routerName, err)
	}

	statusMsg := "berhasil di-hidupkan"
	if command == "matikan" {
		statusMsg = "berhasil di-disable"
	}

	return fmt.Sprintf("*Lokasi*: %s\n*User*: %s\n*Status*: %s", routerName, comment, statusMsg)
}

// 4. TAMBAH / EDIT CLIENT (BINDING & QUEUE)
func (m *Manager) EditAtauTambahClient(ip, limit, nama string) string {
	// 1. Tentukan Router dan Kunci
	client, mu, routerName, err := m.getTargetRouter(ip)
	if err != nil {
		return err.Error()
	}

	limitBytes := formatLimitToBytes(limit)

	// KUNCI PINTU
	mu.Lock()
	// Kita akan unlock manual sebelum return CariAlamatIP

	// A. BINDING
	reply, _ := client.Run("/ip/hotspot/ip-binding/print", "?address="+ip)

	if reply == nil || len(reply.Re) == 0 {
		// Tambah Baru
		_, err = client.Run("/ip/hotspot/ip-binding/add",
			"=address="+ip,
			"=type=bypassed",
			"=disabled=no",
			"=comment="+nama,
		)
	} else {
		// Edit Existing
		id := reply.Re[0].Map[".id"]
		_, err = client.Run("/ip/hotspot/ip-binding/set", "=.id="+id, "=comment="+nama)
	}

	if err != nil {
		mu.Unlock() // Unlock jika error
		return fmt.Sprintf("Gagal proses binding di %s: %v", routerName, err)
	}

	// B. QUEUE
	qReply, _ := client.Run("/queue/simple/print", "?target="+ip+"/32")

	if qReply == nil || len(qReply.Re) == 0 {
		// Tambah Queue
		_, err = client.Run("/queue/simple/add",
			"=name="+nama,
			"=target="+ip+"/32",
			"=max-limit="+limitBytes,
			"=comment="+nama+" - nd",
		)
	} else {
		// Edit Queue
		qID := qReply.Re[0].Map[".id"]
		_, err = client.Run("/queue/simple/set",
			"=.id="+qID,
			"=name="+nama,
			"=max-limit="+limitBytes,
			"=comment="+nama+" - nd",
		)
	}

	if err != nil {
		mu.Unlock() // Unlock jika error
		return fmt.Sprintf("Binding sukses di %s, tapi Queue gagal: %v", routerName, err)
	}

	// BUKA KUNCI SEBELUM PANGGIL FUNGSI LAIN
	mu.Unlock()

	// Return cari alamat IP (Fungsi ini aman dipanggil karena sudah di-unlock)
	return m.CariAlamatIP(ip)
}

// 5. PUTUS CLIENT
func (m *Manager) PutusClient(ip string) string {
	client, mu, routerName, err := m.getTargetRouter(ip)
	if err != nil {
		return err.Error()
	}

	mu.Lock()
	defer mu.Unlock()

	bReply, _ := client.Run("/ip/hotspot/ip-binding/print", "?address="+ip)
	if bReply != nil && len(bReply.Re) > 0 {
		id := bReply.Re[0].Map[".id"]
		client.Run("/ip/hotspot/ip-binding/remove", "=.id="+id)
	} else {
		return fmt.Sprintf("IP tidak ditemukan di binding %s.", routerName)
	}

	qReply, _ := client.Run("/queue/simple/print", "?target="+ip+"/32")
	if qReply != nil && len(qReply.Re) > 0 {
		id := qReply.Re[0].Map[".id"]
		client.Run("/queue/simple/remove", "=.id="+id)
	}

	return fmt.Sprintf("Client %s berhasil diputus dari %s.", ip, routerName)
}

// 6. EDIT LIMIT
func (m *Manager) EditLimit(ip, limit string) string {
	// 1. Tentukan Router
	client, mu, routerName, err := m.getTargetRouter(ip)
	if err != nil {
		return err.Error()
	}

	limitBytes := formatLimitToBytes(limit)

	// KUNCI PINTU
	mu.Lock()
	// Manual unlock sebelum return CariAlamatIP

	// Cari Queue
	qReply, err := client.Run("/queue/simple/print", "?target="+ip+"/32")

	if err != nil || qReply == nil || len(qReply.Re) == 0 {
		mu.Unlock()
		return fmt.Sprintf("IP %s belum dilimit atau tidak ditemukan di %s.", ip, routerName)
	}

	id := qReply.Re[0].Map[".id"]
	name := qReply.Re[0].Map["name"]

	// Eksekusi Update
	_, err = client.Run("/queue/simple/set",
		"=.id="+id,
		"=name="+name,
		"=max-limit="+limitBytes,
	)

	if err != nil {
		mu.Unlock()
		return fmt.Sprintf("Gagal edit limit di %s: %v", routerName, err)
	}

	// BUKA KUNCI
	mu.Unlock()

	// Return hasil terbaru
	return m.CariAlamatIP(ip)
}
