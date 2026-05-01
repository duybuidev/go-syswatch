package main
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)
type MonitorData struct {
	CPUUsage   float64
	MemUsedPct float64
	Containers []types.Container
}
func sendDiscordAlert(webhookURL, message string) {
	payload := map[string]string{"content": message}
	jsonPayload, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("❌ Lỗi kết nối Discord API: %v\n", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("📡 Discord response status: %d\n", resp.StatusCode)
}
func main() {
	discordWebhook := os.Getenv("DISCORD_WEBHOOK_URL")
	if discordWebhook == "" {
		log.Println("⚠️ CẢNH BÁO: DISCORD_WEBHOOK_URL đang trống!")
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("❌ Lỗi kết nối Docker: %v", err)
	}
	defer cli.Close()
	log.Println("🚀 Go-SysWatch started and monitoring...")
	dataChannel := make(chan MonitorData)
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ctx := context.Background()
			cpuPercents, _ := cpu.Percent(0, false)
			vMem, _ := mem.VirtualMemory()
			containers, _ := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
			dataChannel <- MonitorData{CPUUsage: cpuPercents[0], MemUsedPct: vMem.UsedPercent, Containers: containers}
		}
	}()
	alertedContainers := make(map[string]bool)
	for data := range dataChannel {
		for _, ctr := range data.Containers {
			if ctr.State != "running" {
				if !alertedContainers[ctr.ID] && discordWebhook != "" {
					msg := fmt.Sprintf("🚨 **ALERT:** Container `%s` is **%s**!", ctr.Names[0], ctr.State)
					sendDiscordAlert(discordWebhook, msg)
					log.Printf("⚠️ Đã gửi alert cho: %s\n", ctr.Names[0])
					alertedContainers[ctr.ID] = true 
				}
			} else {
				alertedContainers[ctr.ID] = false 
			}
		}
	}
}
