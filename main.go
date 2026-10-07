package main

import (
	"bytes"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/joho/godotenv"
	"golang.org/x/image/font"
)

const (
	canvasW  = 1280
	canvasH  = 720
	rectW    = 760
	rectH    = 428
	rectX    = 240
	rectY    = 62
	radius   = 15
	cacheDir = "thumbnails"
)

var (
	fontBold     *truetype.Font
	fontLight    *truetype.Font
	white        = color.RGBA{255, 255, 255, 255}
	maskImage    *image.Alpha
	lightOverlay image.Image
	client       = &http.Client{Timeout: 15 * time.Second}
)

// Dummy Track struct for prototype
type Track struct {
	ID       string
	Title    string
	Source   string
	Duration int
	Artwork  string
}

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it. Make sure it exists.")
	}

	// Load Fonts
	if b, err := os.ReadFile(filepath.Join("fonts", "Raleway-Bold.ttf")); err == nil {
		fontBold, _ = truetype.Parse(b)
	} else {
		log.Println("Warning: Raleway-Bold.ttf not found in fonts/ folder")
	}
	if b, err := os.ReadFile(filepath.Join("fonts", "Inter-Light.ttf")); err == nil {
		fontLight, _ = truetype.Parse(b)
	} else {
		log.Println("Warning: Inter-Light.ttf not found in fonts/ folder")
	}

	// Generate rounded mask
	maskImage = roundedRectMask(rectW, rectH, radius)

	// Load Light Leak Overlay for Prism Effect
	if file, err := os.Open("light_leak.png"); err == nil {
		defer file.Close()
		if img, err := png.Decode(file); err == nil {
			lightOverlay = imaging.Resize(img, canvasW, canvasH, imaging.Lanczos)
		}
	} else {
		log.Println("Warning: light_leak.png not found, generating without overlay")
	}
}

func main() {
	botToken := os.Getenv("BOT_TOKEN")
	imageUrl := os.Getenv("IMAGE")

	if botToken == "" {
		log.Fatal("BOT_TOKEN is missing in .env")
	}
	if imageUrl == "" {
		log.Fatal("IMAGE is missing in .env")
	}

	fmt.Println("=====================================")
	fmt.Printf("Bot Started Successfully!\nToken Loaded: %s***\n", botToken[:4])
	fmt.Println("=====================================")
	fmt.Println("Generating Thumbnail...")

	// Create dummy track data using the URL from .env
	track := &Track{
		ID:       "test_track_001",
		Title:    "Prototype Testing Track",
		Source:   "YouTube",
		Duration: 215, // 3:35
		Artwork:  imageUrl,
	}

	// Generate the image
	path, err := Generate(track)
	if err != nil {
		log.Fatalf("Error generating thumbnail: %v", err)
	}

	fmt.Printf("Success! Thumbnail saved at: %s\n", path)
}

func Generate(track *Track) (string, error) {
	if track == nil || track.Artwork == "" {
		return "", fmt.Errorf("no artwork available")
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}

	output := filepath.Join(cacheDir, track.ID+".jpg")
	
	src, err := downloadImage(track.Artwork)
	if err != nil {
		return "", err
	}

	// 1. Stretch
	stretched := imaging.Resize(src, canvasW, canvasH, imaging.Lanczos)

	// 2. Blur & Dim background
	bg := imaging.Blur(stretched, 25)
	bg = multiplyBrightness(bg, 0.40)

	// 3. Zoom
	zoomed := imaging.CropCenter(stretched, 1040, 585)

	// 4. Foreground
	fg := imaging.Fill(zoomed, rectW, rectH, imaging.Center, imaging.Lanczos)

	// 5. Canvas
	canvas := image.NewRGBA(bg.Bounds())
	draw.Draw(canvas, canvas.Bounds(), bg, image.Point{}, draw.Src)

	// 6. Draw foreground mask
	draw.DrawMask(canvas, image.Rect(rectX, rectY, rectX+rectW, rectY+rectH), fg, image.Point{}, maskImage, image.Point{}, draw.Over)

	// 6.5 Apply Light Prism Overlay (If available)
	if lightOverlay != nil {
		draw.Draw(canvas, canvas.Bounds(), lightOverlay, image.Point{}, draw.Over)
	}

	// 7. Text
	source := track.Source
	title := sanitizeText(track.Title)

	if fontLight != nil && fontBold != nil {
		drawText(canvas, fontLight, 30, 50, 560, source+" | by SiloMusic")
		drawText(canvas, fontBold, 30, 50, 600, title)
		drawText(canvas, fontBold, 30, 40, 650, "0:01")
		drawText(canvas, fontBold, 30, 1185, 650, formatDuration(track.Duration))
	}

	drawLine(canvas, 140, 1160, 670, 5)

	f, err := os.Create(output)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := jpeg.Encode(f, canvas, &jpeg.Options{Quality: 92}); err != nil {
		os.Remove(output)
		return "", err
	}

	return output, nil
}

func downloadImage(url string) (image.Image, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	return img, err
}

func multiplyBrightness(img *image.NRGBA, factor float64) *image.NRGBA {
	out := image.NewNRGBA(img.Rect)
	for i := 0; i < len(img.Pix); i += 4 {
		out.Pix[i] = uint8(float64(img.Pix[i]) * factor)
		out.Pix[i+1] = uint8(float64(img.Pix[i+1]) * factor)
		out.Pix[i+2] = uint8(float64(img.Pix[i+2]) * factor)
		out.Pix[i+3] = img.Pix[i+3]
	}
	return out
}

func roundedRectMask(w, h, r int) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	rf := float64(r)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			alpha := uint8(255)
			if fx < rf && fy < rf {
				alpha = calculateAntiAliasedAlpha(math.Hypot(rf-fx, rf-fy), rf)
			} else if fx >= float64(w)-rf && fy < rf {
				alpha = calculateAntiAliasedAlpha(math.Hypot(fx-(float64(w)-rf), rf-fy), rf)
			} else if fx < rf && fy >= float64(h)-rf {
				alpha = calculateAntiAliasedAlpha(math.Hypot(rf-fx, fy-(float64(h)-rf)), rf)
			} else if fx >= float64(w)-rf && fy >= float64(h)-rf {
				alpha = calculateAntiAliasedAlpha(math.Hypot(fx-(float64(w)-rf), fy-(float64(h)-rf)), rf)
			}
			if alpha > 0 {
				mask.SetAlpha(x, y, color.Alpha{A: alpha})
			}
		}
	}
	return mask
}

func calculateAntiAliasedAlpha(dist, radius float64) uint8 {
	if dist <= radius-0.5 {
		return 255
	}
	if dist >= radius+0.5 {
		return 0
	}
	return uint8(((radius + 0.5) - dist) * 255)
}

func drawText(dst draw.Image, f *truetype.Font, size float64, x, y int, text string) {
	ctx := freetype.NewContext()
	ctx.SetDPI(72)
	ctx.SetFont(f)
	ctx.SetFontSize(size)
	ctx.SetClip(dst.Bounds())
	ctx.SetDst(dst)
	ctx.SetSrc(image.NewUniform(white))
	ctx.SetHinting(font.HintingFull)
	pt := freetype.Pt(x, y+int(size))
	_, _ = ctx.DrawString(text, pt)
}

func drawLine(dst draw.Image, x1, x2, y, width int) {
	half := width / 2
	rect := image.Rect(x1, y-half, x2, y-half+width)
	draw.Draw(dst, rect, image.NewUniform(white), image.Point{}, draw.Over)
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func sanitizeText(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 32 && r < 127) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func formatDuration(sec int) string {
	if sec <= 0 {
		return "0:00"
	}
	m := sec / 60
	s := sec % 60
	return strconv.Itoa(m) + ":" + fmt.Sprintf("%02d", s)
}
