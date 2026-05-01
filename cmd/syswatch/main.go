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
	if err == nil {
		resp.Body.Close()
	}
}

func main() {
	discordWebhook := os.Getenv("DISCORD_WEBHOOK_URL")

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("❌ Lỗi kết nối Docker: %v", err)
	}
	defer cli.Close()

	log.Println("🚀 Go-SysWatch v2 Started! Monitoring Up/Down events...")
	dataChannel := make(chan MonitorData)

	// Map lưu trạng thái cuối cùng của từng container (ID -> State)
	containerLastState := make(map[string]string)

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ctx := context.Background()
			cpuPercents, _ := cpu.Percent(0, false)
			vMem, _ := mem.VirtualMemory()
			containers, _ := cli.ContainerList(ctx, types.ContainerListOptions{All: true})

			dataChannel <- MonitorData{
				CPUUsage:   cpuPercents[0],
				MemUsedPct: vMem.UsedPercent,
				Containers: containers,
			}
		}
	}()

	for data := range dataChannel {
		for _, ctr := range data.Containers {
			containerName := ctr.Names[0]
			currentState := ctr.State
			lastState, exists := containerLastState[ctr.ID]

			// Nếu đây là lần đầu quét thấy container này, chỉ ghi nhớ rồi bỏ qua (tránh spam lúc mới bật app)
			if !exists {
				containerLastState[ctr.ID] = currentState
				continue
			}

			// LOGIC 1: Nếu trạng thái đổi từ "không chạy" sang "running"
			if lastState != "running" && currentState == "running" {
				msg := fmt.Sprintf("✅ **PHỤC HỒI:** Container `%s` đã **BẬT** trở lại! (Status: %s)", containerName, currentState)
				sendDiscordAlert(discordWebhook, msg)
				log.Printf("🔔 Alert: %s is Up\n", containerName)
			}

			// LOGIC 2: Nếu trạng thái đổi từ "running" sang "không chạy"
			if lastState == "running" && currentState != "running" {
				msg := fmt.Sprintf("🚨 **CẢNH BÁO:** Container `%s` vừa bị **SẬP**! (Status: %s)", containerName, currentState)
				sendDiscordAlert(discordWebhook, msg)
				log.Printf("🔔 Alert: %s is Down\n", containerName)
			}

			// Cập nhật lại trạng thái cuối cùng
			containerLastState[ctr.ID] = currentState
		}
	}
}
