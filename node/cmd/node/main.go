package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Push struct {
	Token         string            `json:"token"`
	Type          string            `json:"type"`
	Hostname      string            `json:"hostname"`
	PublicIP      string            `json:"public_ip"`
	OS            string            `json:"os"`
	Timestamp     time.Time         `json:"timestamp"`
	CPUPercent    float64           `json:"cpu_percent"`
	RAMPercent    float64           `json:"ram_percent"`
	DiskPercent   float64           `json:"disk_percent"`
	RxBps         int64             `json:"rx_bps"`
	TxBps         int64             `json:"tx_bps"`
	RxBytes       int64             `json:"rx_bytes"`
	TxBytes       int64             `json:"tx_bytes"`
	OnlineUsers   int               `json:"online_users"`
	OpenVPNStatus string            `json:"openvpn_status"`
	L2TPStatus    string            `json:"l2tp_status"`
	IKEv2Status   string            `json:"ikev2_status"`
	Services      map[string]string `json:"services"`
}

type Task struct {
	ID      int64           `json:"id"`
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload_json"`
}

type TaskPollResponse struct {
	OK    bool   `json:"ok"`
	Tasks []Task `json:"tasks"`
}

func main() {
	panel := strings.TrimRight(getenv("PANEL_URL", "http://127.0.0.1:8080"), "/")
	token := getenv("NODE_TOKEN", "")
	if token == "" {
		log.Fatal("NODE_TOKEN is required")
	}
	intervalSeconds, _ := strconv.Atoi(getenv("NODE_INTERVAL", "10"))
	if intervalSeconds < 3 {
		intervalSeconds = 3
	}
	interval := time.Duration(intervalSeconds) * time.Second
	client := &http.Client{Timeout: 20 * time.Second}

	lastRx, lastTx := netBytes()
	lastAt := time.Now()
	for {
		nowRx, nowTx := netBytes()
		now := time.Now()
		dt := now.Sub(lastAt).Seconds()
		if dt <= 0 {
			dt = interval.Seconds()
		}
		host, _ := os.Hostname()
		services := map[string]string{
			"openvpn": serviceStatus("openvpn"),
			"l2tp":    serviceStatus("xl2tpd"),
			"ikev2":   serviceStatus("strongswan"),
			"ssh":     serviceStatus("ssh"),
		}
		push := Push{
			Token:         token,
			Type:          "status",
			Hostname:      host,
			PublicIP:      firstIP(),
			OS:            runtime.GOOS,
			Timestamp:     now.UTC(),
			CPUPercent:    cpuPercent(),
			RAMPercent:    memPercent(),
			DiskPercent:   diskPercent("/"),
			RxBytes:       nowRx,
			TxBytes:       nowTx,
			RxBps:         int64(float64(nowRx-lastRx) / dt),
			TxBps:         int64(float64(nowTx-lastTx) / dt),
			OnlineUsers:   0,
			OpenVPNStatus: services["openvpn"],
			L2TPStatus:    services["l2tp"],
			IKEv2Status:   services["ikev2"],
			Services:      services,
		}
		postJSON(client, panel+"/api/node/push", token, push)
		pollTasks(client, panel, token)
		lastRx, lastTx, lastAt = nowRx, nowTx, now
		time.Sleep(interval)
	}
}

func pollTasks(client *http.Client, panel, token string) {
	req, _ := http.NewRequest(http.MethodPost, panel+"/api/node/tasks/poll", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Node-Token", token)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("task poll failed: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		log.Printf("task poll status: %s", resp.Status)
		return
	}
	var out TaskPollResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		log.Printf("task poll decode: %v", err)
		return
	}
	for _, task := range out.Tasks {
		status, result, errText := executeTask(task)
		complete := map[string]any{"status": status, "result_json": result, "error": errText}
		postJSON(client, fmt.Sprintf("%s/api/node/tasks/%d/complete", panel, task.ID), token, complete)
	}
}

func executeTask(task Task) (string, map[string]any, string) {
	var payload map[string]any
	_ = json.Unmarshal(task.Payload, &payload)
	switch task.Action {
	case "agent.status":
		return "succeeded", map[string]any{"message": "agent alive", "time": time.Now().UTC()}, ""
	case "service.status":
		service := normalizeService(fmt.Sprint(payload["service"]))
		if service == "" {
			return "failed", map[string]any{}, "invalid service"
		}
		return "succeeded", map[string]any{"service": service, "status": serviceStatus(service)}, ""
	case "service.restart", "service.reload":
		service := normalizeService(fmt.Sprint(payload["service"]))
		if service == "" {
			return "failed", map[string]any{}, "invalid service"
		}
		verb := "restart"
		if task.Action == "service.reload" {
			verb = "reload"
		}
		cmd := exec.Command("systemctl", verb, service)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "failed", map[string]any{"service": service, "output": string(out)}, err.Error()
		}
		return "succeeded", map[string]any{"service": service, "output": string(out), "status": serviceStatus(service)}, ""
	default:
		return "failed", map[string]any{}, "unsupported action"
	}
}

func normalizeService(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	switch s {
	case "openvpn", "openvpn@server", "openvpn-server@server":
		return "openvpn"
	case "l2tp", "xl2tpd":
		return "xl2tpd"
	case "ikev2", "ipsec", "strongswan", "strongswan-starter":
		return "strongswan"
	case "ssh", "sshd", "ssh-tunnel", "dropbear":
		return "ssh"
	default:
		return ""
	}
}

func postJSON(client *http.Client, url, token string, v any) {
	b, _ := json.Marshal(v)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Node-Token", token)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("post %s failed: %v", url, err)
		return
	}
	_ = resp.Body.Close()
	log.Printf("post %s: %s", url, resp.Status)
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func firstIP() string {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP == nil || ipNet.IP.To4() == nil {
				continue
			}
			return ipNet.IP.String()
		}
	}
	return ""
}

func serviceStatus(service string) string {
	// Map logical names to systemd unit names
	unitName := service
	switch service {
	case "ssh":
		// Try sshd first (most distros), fallback to ssh (Debian/Ubuntu)
		out, err := exec.Command("systemctl", "is-active", "sshd").Output()
		if err == nil {
			status := strings.TrimSpace(string(out))
			if status == "active" {
				return "running"
			}
		}
		unitName = "ssh"
	case "openvpn":
		// Try openvpn@server first, fallback to openvpn
		out, err := exec.Command("systemctl", "is-active", "openvpn@server").Output()
		if err == nil {
			status := strings.TrimSpace(string(out))
			if status == "active" {
				return "running"
			}
		}
		unitName = "openvpn"
	}
	out, err := exec.Command("systemctl", "is-active", unitName).Output()
	if err != nil {
		return "stopped"
	}
	status := strings.TrimSpace(string(out))
	switch status {
	case "active":
		return "running"
	case "inactive", "dead":
		return "stopped"
	case "failed":
		return "failed"
	default:
		return status
	}
}

func cpuPercent() float64 {
	idle1, total1 := readCPU()
	time.Sleep(180 * time.Millisecond)
	idle2, total2 := readCPU()
	idle := float64(idle2 - idle1)
	total := float64(total2 - total1)
	if total <= 0 {
		return 0
	}
	return round2((1 - idle/total) * 100)
}

func readCPU() (idle, total uint64) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0
	}
	fields := strings.Fields(strings.SplitN(string(b), "\n", 2)[0])
	for i, field := range fields[1:] {
		v, _ := strconv.ParseUint(field, 10, 64)
		total += v
		if i == 3 || i == 4 {
			idle += v
		}
	}
	return idle, total
}

func memPercent() float64 {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	vals := map[string]float64{}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			key := strings.TrimSuffix(fields[0], ":")
			vals[key], _ = strconv.ParseFloat(fields[1], 64)
		}
	}
	total := vals["MemTotal"]
	available := vals["MemAvailable"]
	if total <= 0 {
		return 0
	}
	return round2((total - available) / total * 100)
}

func diskPercent(mount string) float64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(mount, &stat); err != nil {
		return 0
	}
	total := float64(stat.Blocks)
	free := float64(stat.Bavail)
	if total <= 0 {
		return 0
	}
	return round2((total - free) / total * 100)
}

func netBytes() (rx, tx int64) {
	b, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.Split(line, ":")
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}
		r, _ := strconv.ParseInt(fields[0], 10, 64)
		t, _ := strconv.ParseInt(fields[8], 10, 64)
		rx += r
		tx += t
	}
	return rx, tx
}

func round2(v float64) float64 {
	n, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", v), 64)
	return n
}
