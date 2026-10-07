// cmd/demo/main.go — SiloMusic_test /play Now Playing Rich Message UI Demo

package main

import (
	"fmt"
	"html"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	_ "github.com/joho/godotenv/autoload" // auto-loads .env from CWD
	tg "github.com/amarnathcjd/gogram/telegram"
)

// ── Rich Message custom TL types ─────────────────────────────────────────────

type richButtonStyle struct {
	Flags     int32 `tl:"flag"`
	BgPrimary bool  `tl:"flag:0,encoded_in_bitflags"`
	BgDanger  bool  `tl:"flag:1,encoded_in_bitflags"`
	BgSuccess bool  `tl:"flag:2,encoded_in_bitflags"`
	Link      bool  `tl:"flag:3,encoded_in_bitflags"`
}

func (*richButtonStyle) CRC() uint32    { return 0x03c610bd }
func (*richButtonStyle) FlagIndex() int { return 0 }

type inlineButtonTypeCallback struct {
	Flags            int32  `tl:"flag"`
	RequiresPassword bool   `tl:"flag:0,encoded_in_bitflags"`
	Data             []byte
}

func (*inlineButtonTypeCallback) CRC() uint32    { return 0x2955bc38 }
func (*inlineButtonTypeCallback) FlagIndex() int { return 0 }

type pageButton struct {
	Flags int32            `tl:"flag"`
	Text  tg.RichText
	Type  any
	Style *richButtonStyle `tl:"flag:0"`
}

func (*pageButton) CRC() uint32    { return 0x692a5488 }
func (*pageButton) FlagIndex() int { return 0 }

type pageBlockButtonRow struct {
	Flags       int32 `tl:"flag"`
	AlignLeft   bool  `tl:"flag:0,encoded_in_bitflags"`
	AlignCenter bool  `tl:"flag:1,encoded_in_bitflags"`
	AlignRight  bool  `tl:"flag:2,encoded_in_bitflags"`
	Buttons     []*pageButton
}

func (*pageBlockButtonRow) CRC() uint32          { return 0x6d640318 }
func (*pageBlockButtonRow) FlagIndex() int       { return 0 }
func (*pageBlockButtonRow) ImplementsPageBlock() {} // Mandatory for Toggle Block

// ── Demo static data ──────────────────────────────────────────────────────────

const (
	demoTitle     = "Tere Liye (Veer-Zaara)"
	demoURL       = "https://youtu.be/pMsDaE4IXNM"
	demoDuration  = 275 // 4:35
	demoRequester = `<a href="tg://user?id=123456">Demo User</a>`
	demoThumb     = "https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg"
)

var demoQueue = []string{
	"Ae Dil Hai Mushkil — Arijit Singh",
	"Channa Mereya — Arijit Singh",
	"Tera Ban Jaunga — Akhil",
}

// ── Per-chat demo state ───────────────────────────────────────────────────────

var (
	mu            sync.Mutex
	pausedByChat  = map[int64]bool{}
	cardMsgByChat = map[int64]*tg.NewMessage{}
)

func isPaused(chatID int64) bool {
	mu.Lock(); defer mu.Unlock()
	return pausedByChat[chatID]
}

func setPaused(chatID int64, v bool) {
	mu.Lock(); defer mu.Unlock()
	pausedByChat[chatID] = v
}

func setCard(chatID int64, msg *tg.NewMessage) {
	mu.Lock(); defer mu.Unlock()
	cardMsgByChat[chatID] = msg
}

func getCard(chatID int64) *tg.NewMessage {
	mu.Lock(); defer mu.Unlock()
	return cardMsgByChat[chatID]
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func fmtDuration(sec int) string {
	if sec <= 0 {
		return "0:00"
	}
	return strconv.Itoa(sec/60) + ":" + fmt.Sprintf("%02d", sec%60)
}

// Sleek Progress Bar Style
func progressBar(pos, dur int) string {
	const width = 13
	if dur <= 0 {
		return "◉" + strings.Repeat("━", width)
	}
	filled := width * pos / dur
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("━", filled) + "◉" + strings.Repeat("━", width-filled)
}

func progressText(pos, dur int) string {
	return fmtDuration(pos) + " " + progressBar(pos, dur) + " " + fmtDuration(dur)
}

var requesterRe = regexp.MustCompile(`<a\s+href="([^"]+)">([^<]+)</a>`)

func parseRequesterRichText(raw string) tg.RichText {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return &tg.TextPlain{Text: "User"}
	}
	if m := requesterRe.FindStringSubmatch(raw); len(m) == 3 {
		return &tg.TextURL{
			Text: &tg.TextPlain{Text: html.UnescapeString(m[2])},
			URL:  m[1],
		}
	}
	return &tg.TextPlain{Text: html.UnescapeString(raw)}
}

// ── Rich Message builder ──────────────────────────────────────────────────────

func buildDemoCard(
	client *tg.Client,
	chatID int64,
	paused bool,
	pos int,
) (*tg.RichBuilder, error) {
	// 1. Heading
	headingBlock := &tg.PageBlockHeading2{
		Text: &tg.TextBold{
			Text: &tg.TextUnderline{
				Text: &tg.TextPlain{Text: "🎵 Now playing..."},
			},
		},
	}

	// 2. Details Box
	titleRich := tg.RichText(&tg.TextURL{
		Text: &tg.TextPlain{Text: demoTitle},
		URL:  demoURL,
	})

	tableConcat := &tg.TextConcat{
		Texts: []tg.RichText{
			&tg.TextBold{Text: &tg.TextPlain{Text: "Title: "}},
			titleRich,
			&tg.TextPlain{Text: "\n"},
			&tg.TextBold{Text: &tg.TextPlain{Text: "Duration: "}},
			&tg.TextPlain{Text: fmtDuration(demoDuration) + "\n"},
			&tg.TextBold{Text: &tg.TextPlain{Text: "Requested by: "}},
			parseRequesterRichText(demoRequester),
		},
	}

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

	// 3. Photo Block
	var photoBlock tg.PageBlock
	var ipo *tg.InputPhotoObj

	media, err := client.GetSendableMedia(demoThumb, &tg.MediaMetadata{Inline: true})
	if err == nil {
		if mp, ok := media.(*tg.InputMediaPhoto); ok {
			if p, ok := mp.ID.(*tg.InputPhotoObj); ok {
				ipo = p
				photoBlock = &tg.PageBlockPhoto{
					PhotoID: ipo.ID,
					Caption: &tg.PageCaption{
						Text:   &tg.TextEmpty{},
						Credit: &tg.TextEmpty{},
					},
				}
			}
		}
	}

	// 4. Create Buttons (Controls & Queue)
	var pauseBtn *pageButton
	if paused {
		pauseBtn = &pageButton{
			Text:  &tg.TextPlain{Text: "▷ Resume"},
			Type:  &inlineButtonTypeCallback{Data: []byte("demo:resume")},
			Style: &richButtonStyle{BgSuccess: true},
		}
	} else {
		pauseBtn = &pageButton{
			Text:  &tg.TextPlain{Text: "II Pause"},
			Type:  &inlineButtonTypeCallback{Data: []byte("demo:pause")},
			Style: &richButtonStyle{BgSuccess: true},
		}
	}

	controlsRow := &pageBlockButtonRow{
		Buttons: []*pageButton{
			{
				Text:  &tg.TextPlain{Text: "↺ Replay"},
				Type:  &inlineButtonTypeCallback{Data: []byte("demo:replay")},
				Style: &richButtonStyle{BgPrimary: true},
			},
			pauseBtn,
			{
				Text:  &tg.TextPlain{Text: "» Skip"},
				Type:  &inlineButtonTypeCallback{Data: []byte("demo:skip")},
				Style: &richButtonStyle{BgPrimary: true},
			},
		},
	}

	queueRow := &pageBlockButtonRow{
		Buttons: []*pageButton{
			{
				Text:  &tg.TextPlain{Text: fmt.Sprintf("☰ Queue · %d", len(demoQueue))},
				Type:  &inlineButtonTypeCallback{Data: []byte("demo:queue")},
				Style: &richButtonStyle{BgSuccess: true},
			},
		},
	}

	// 5. TOGGLE BLOCK: Magic happens here!
	toggleBlock := &tg.PageBlockDetails{
		Open: false, // Default band rahega sabke liye
		Title: &tg.TextPlain{
			// Progress bar hi iska title hai
			Text: progressText(pos, demoDuration),
		},
		Blocks: []tg.PageBlock{
			controlsRow,
			queueRow,
		},
	}

	// ── Assemble ──────────────────────────────────────────────────────────────
	rb := tg.NewRichMessage()
	rb.AddBlock(headingBlock)
	rb.AddBlock(tableBlock)
	if photoBlock != nil && ipo != nil {
		rb.AddBlock(photoBlock)
		rb.AttachPhoto(ipo)
	}
	// Direct toggleBlock lagaya, buttons iske andar hain
	rb.AddBlock(toggleBlock)

	return rb, nil
}

// ── Send / update helpers ─────────────────────────────────────────────────────

func sendDemoCard(client *tg.Client, chatID int64) {
	paused := isPaused(chatID)
	
	// Demo position = 149 seconds (2:29)
	rb, err := buildDemoCard(client, chatID, paused, 149)
	if err != nil {
		log.Printf("[demo] buildDemoCard: %v", err)
		return
	}

	existing := getCard(chatID)
	if existing != nil {
		edited, err := client.EditRich(chatID, int32(existing.ID), rb)
		if err == nil {
			setCard(chatID, edited)
			return
		}
		log.Printf("[demo] EditRich failed (%v), sending new card", err)
	}

	sent, err := client.SendRich(chatID, rb)
	if err != nil {
		log.Printf("[demo] SendRich failed: %v", err)
		return
	}
	setCard(chatID, sent)
}

// ── Handler: /start ───────────────────────────────────────────────────────────

func handleStart(m *tg.NewMessage) error {
	chatID := m.ChannelID()
	setPaused(chatID, false)
	sendDemoCard(m.Client, chatID)
	return tg.ErrEndGroup
}

// ── Handler: callback queries ─────────────────────────────────────────────────

func handleCallback(cb *tg.CallbackQuery) error {
	data    := cb.DataString()
	chatID  := cb.ChannelID()
	alertOpt := &tg.CallbackOptions{Alert: true}

	switch data {
	case "demo:pause":
		setPaused(chatID, true)
		cb.Answer("⏸ Paused", &tg.CallbackOptions{Alert: false})
		sendDemoCard(cb.Client, chatID)

	case "demo:resume":
		setPaused(chatID, false)
		cb.Answer("▶ Resumed", &tg.CallbackOptions{Alert: false})
		sendDemoCard(cb.Client, chatID)

	case "demo:replay":
		cb.Answer("↺ Replaying: "+demoTitle, alertOpt)

	case "demo:skip":
		setPaused(chatID, false)
		cb.Answer("» Skipped! (demo — no next track)", alertOpt)
		sendDemoCard(cb.Client, chatID)

	case "demo:queue":
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("☰ Queue (%d tracks):\n\n", len(demoQueue)))
		for i, t := range demoQueue {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, t))
		}
		cb.Answer(strings.TrimRight(sb.String(), "\n"), alertOpt)

	default:
		cb.Answer("")
	}

	return tg.ErrEndGroup
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	apiIDStr := os.Getenv("API_ID")
	apiHash  := os.Getenv("API_HASH")
	token    := os.Getenv("TOKEN")
	if token == "" {
		token = os.Getenv("BOT_TOKEN")
	}

	if apiIDStr == "" || apiHash == "" || token == "" {
		log.Fatal("❌  API_ID, API_HASH, and TOKEN must all be set in .env")
	}

	apiID64, err := strconv.ParseInt(apiIDStr, 10, 32)
	if err != nil {
		log.Fatalf("❌  Invalid API_ID: %v", err)
	}

	client, err := tg.NewClient(tg.ClientConfig{
		AppID:    int32(apiID64),
		AppHash:  apiHash,
		LogLevel: tg.LogError,
		Session:  "demo.session",
	})
	if err != nil {
		log.Fatalf("❌  NewClient: %v", err)
	}

	if err := client.LoginBot(token); err != nil {
		log.Fatalf("❌  LoginBot: %v", err)
	}

	me, err := client.GetMe()
	if err != nil {
		log.Fatalf("❌  GetMe: %v", err)
	}

	fmt.Printf("\n✅  Demo bot connected as @%s\n", me.Username)
	fmt.Println("    Send /start and click the Progress Bar to reveal controls instantly!")

	client.SetCommandPrefixes("/")
	client.AddCommandHandler("start", func(m *tg.NewMessage) error {
		return handleStart(m)
	})

	for _, action := range []string{"pause", "resume", "replay", "skip", "queue"} {
		a := action 
		client.AddCallbackHandler("demo:"+a, func(cb *tg.CallbackQuery) error {
			return handleCallback(cb)
		})
	}

	client.Idle()
}
