package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"
	tg "github.com/amarnathcjd/gogram/telegram"
)

func buildSimpleCard() *tg.RichBuilder {
	// 1. Heading Block
	headingBlock := &tg.PageBlockHeading2{
		Text: &tg.TextBold{
			Text: &tg.TextUnderline{
				Text: &tg.TextPlain{Text: "🎵 Now playing..."},
			},
		},
	}

	// 2. Table Block with only progress text
	tableConcat := &tg.TextPlain{Text: "2:29 ━━━━━━━◉━━━━━━ 4:35"}

	tableBlock := &tg.PageBlockTable{
		Bordered: true,
		Title:    &tg.TextEmpty{},
		Rows: []*tg.PageTableRow{
			{
				Cells: []*tg.PageTableCell{
					{
						Header: true,
						Text:   tableConcat,
					},
				},
			},
		},
	}

	// Assemble Message
	rb := tg.NewRichMessage()
	rb.AddBlock(headingBlock)
	rb.AddBlock(tableBlock)
	return rb
}

func main() {
	token := os.Getenv("TOKEN")
	if token == "" {
		token = os.Getenv("BOT_TOKEN")
	}
	apiIDStr := os.Getenv("API_ID")
	apiHash := os.Getenv("API_HASH")

	if apiIDStr == "" || apiHash == "" || token == "" {
		log.Fatal("❌ API_ID, API_HASH, and TOKEN must be set in .env")
	}

	var apiID int32
	fmt.Sscanf(apiIDStr, "%d", &apiID)

	client, err := tg.NewClient(tg.ClientConfig{
		AppID:    apiID,
		AppHash:  apiHash,
		LogLevel: tg.LogError,
		Session:  "simple_demo.session",
	})
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}

	if err := client.LoginBot(token); err != nil {
		log.Fatalf("LoginBot: %v", err)
	}

	client.SetCommandPrefixes("/")
	client.AddCommandHandler("start", func(m *tg.NewMessage) error {
		rb := buildSimpleCard()
		_, err := m.Client.SendRich(m.ChannelID(), rb)
		if err != nil {
			log.Printf("SendRich error: %v", err)
		}
		return tg.ErrEndGroup
	})

	me, _ := client.GetMe()
	fmt.Printf("\n✅ Simple Bot connected as @%s\n", me.Username)
	client.Idle()
}
