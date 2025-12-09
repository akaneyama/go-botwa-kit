package mikrotik

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-routeros/routeros"
)

// Helper: Memilih client berdasarkan IP (Logic dari ambildatapakeAPI)
func (m *Manager) getClientByIP(ip string) (*routeros.Client, error) {
	if strings.HasPrefix(ip, "192.168") || strings.HasPrefix(ip, "123.123") || strings.HasPrefix(ip, "172.16") {
		if m.Client1 == nil {
			return nil, fmt.Errorf("koneksi ke Router 1 terputus")
		}
		return m.Client1, nil
	} else if strings.HasPrefix(ip, "193.168") {
		if m.Client2 == nil {
			return nil, fmt.Errorf("koneksi ke Router 2 terputus")
		}
		return m.Client2, nil
	}
	return nil, fmt.Errorf("alamat IP %s tidak dikenali atau salah", ip)
}

// Helper: Format Speed (Mbps/Kbps) untuk tampilan (Logic dari formatSpeed)
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
		} else if val >= 1000 {
			return fmt.Sprintf("%d Kbps", val/1000)
		}
		return fmt.Sprintf("%d bps", val)
	}

	return fmt.Sprintf("%s / %s", convert(parts[0]), convert(parts[1]))
}

// Helper: Ubah "10/10" jadi "10000000/10000000" (Logic dari ubahformatmikrotik)
func formatLimitToBytes(queue string) string {
	parts := strings.Split(queue, "/")
	if len(parts) != 2 {
		return queue // return as is if format wrong
	}
	// Asumsi input "10" berarti 10 Mbps (sesuai logic Nodejs + "000000")
	return fmt.Sprintf("%s000000/%s000000", parts[0], parts[1])
}
