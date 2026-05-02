package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

// DÁN LINK WEBHOOK CỦA BẠN VÀO ĐÂY
const discordWebhookURL = "https://discord.com/api/webhooks/xxxx/yyyy"

type DiscordPayload struct {
	Content string `json:"content"`
}

func sendDiscordAlert(message string) {
	if discordWebhookURL == "https://discord.com/api/webhooks/xxxx/yyyy" {
		return // Bỏ qua nếu chưa điền webhook
	}
	payload := DiscordPayload{Content: message}
	jsonValue, _ := json.Marshal(payload)
	resp, err := http.Post(discordWebhookURL, "application/json", bytes.NewBuffer(jsonValue))
	if err == nil {
		resp.Body.Close()
	}
}

// Hàm mới: Điểm danh các container ĐANG CHẠY khi SysWatch vừa bật lên
func checkCurrentState(cli *client.Client) {
	fmt.Println("🔍 Điểm danh hệ thống: Đang kiểm tra các container có sẵn...")
	
	// Lấy danh sách toàn bộ container đang chạy
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		log.Printf("Lỗi khi quét container: %v", err)
		return
	}

	activeCount := len(containers)
	fmt.Printf("✅ Tìm thấy %d containers đang hoạt động.\n", activeCount)
	
	discordMsg := fmt.Sprintf("🛡️ **SysWatch Agent Bootup!**\n✅ Hệ thống hiện đang có **%d** containers hoạt động bình thường.", activeCount)
	
	for _, c := range containers {
		// Tên container trong Docker thường có dấu / ở đầu, ta dùng [1:] để cắt nó đi
		fmt.Printf("   - 🟢 Đang chạy: %s\n", c.Names[0][1:])
	}
	sendDiscordAlert(discordMsg)
	fmt.Println("---------------------------------------------------")
}

func main() {
	fmt.Println("🛡️ SysWatch Agent starting...")

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Fatal: Cannot connect to Docker daemon: %v", err)
	}
	defer cli.Close()

	// 1. GỌI HÀM ĐIỂM DANH TRƯỚC
	checkCurrentState(cli)

	fmt.Println("📡 Listening for NEW container events in real-time...\n")

	// 2. SAU ĐÓ MỚI BẮT ĐẦU ĐỨNG GÁC
	ctx := context.Background()
	msgs, errs := cli.Events(ctx, types.EventsOptions{})

	for {
		select {
		case err := <-errs:
			log.Printf("Error from Docker events: %v\n", err)
		case msg := <-msgs:
			if msg.Type == events.ContainerEventType && msg.Action == "start" {
				name := msg.Actor.Attributes["name"]
				fmt.Printf("🟢 [%s] Container %s STARTED.\n", time.Now().Format("15:04:05"), name)
				sendDiscordAlert(fmt.Sprintf("🟢 **[STARTED]** Container `%s` is up!", name))
			}

			if msg.Type == events.ContainerEventType && msg.Action == "die" {
				name := msg.Actor.Attributes["name"]
				exitCode := msg.Actor.Attributes["exitCode"]
				fmt.Printf("🚨 [%s] Container %s CRASHED! (Code: %s)\n", time.Now().Format("15:04:05"), name, exitCode)
				sendDiscordAlert(fmt.Sprintf("🚨 **[CRASHED]** Container `%s` has DIED! (Exit: %s)", name, exitCode))
			}
		}
	}
}
