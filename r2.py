import os
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry
from dotenv import load_dotenv
from PIL import Image, ImageDraw, ImageEnhance, ImageFilter, ImageFont, ImageOps
from telegram import Update
from telegram.ext import ApplicationBuilder, CommandHandler, ContextTypes

load_dotenv()

BOT_TOKEN = os.getenv("BOT_TOKEN")
API_ID = os.getenv("API_ID")
API_HASH = os.getenv("API_HASH")
DEFAULT_IMAGE_URL = os.getenv("IMAGE", "https://picsum.photos/1280/720")

os.makedirs("cache", exist_ok=True)


class DummyTrack:
    def __init__(self):
        self.id = "sample_track_123"
        self.title = "Anuv Jain - Hoshiyar"
        self.channel_name = "Billiemusic"
        self.view_count = "1.2M"
        self.duration = "3:45"
        self.thumbnail = DEFAULT_IMAGE_URL


class AppleMusicThumbnailGenerator:

    def __init__(self):
        self.fill = (255, 255, 255)
        try:
            self.font_title = ImageFont.truetype(
                "anony/helpers/Raleway-Bold.ttf", 26
            )
            self.font_sub = ImageFont.truetype(
                "anony/helpers/Inter-Light.ttf", 20
            )
            self.font_header = ImageFont.truetype(
                "anony/helpers/Inter-Light.ttf", 16
            )
        except Exception:
            self.font_title = ImageFont.load_default()
            self.font_sub = ImageFont.load_default()
            self.font_header = ImageFont.load_default()

    def save_thumb(self, output_path: str, url: str) -> str:
        headers = {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
        }
        session = requests.Session()
        retries = Retry(total=5, backoff_factor=1, status_forcelist=[500, 502, 503, 504])
        session.mount('https://', HTTPAdapter(max_retries=retries))
        session.mount('http://', HTTPAdapter(max_retries=retries))

        try:
            response = session.get(url, headers=headers, timeout=20)
            if response.status_code == 200:
                with open(output_path, "wb") as f:
                    f.write(response.content)
            else:
                raise Exception(f"HTTP Status: {response.status_code}")
        except Exception:
            img = Image.new("RGB", (1280, 720), (30, 30, 30))
            img.save(output_path)
            
        return output_path

    async def generate(self, song: DummyTrack, size=(1280, 720)) -> str:
        temp = f"cache/temp_{song.id}.jpg"
        output = f"cache/{song.id}.png"

        self.save_thumb(temp, song.thumbnail)

        # 1. Background Blur
        thumb = Image.open(temp).convert("RGBA").resize(size, Image.Resampling.LANCZOS)
        blur = thumb.filter(ImageFilter.GaussianBlur(40))
        image = ImageEnhance.Brightness(blur).enhance(0.30)

        # 2. Window Card Setup (MacBook Style)[span_6](start_span)[span_6](end_span)[span_7](start_span)[span_7](end_span)
        card_width, card_height = 960, 540
        card_x = int((size[0] - card_width) / 2)
        card_y = int((size[1] - card_height) / 2)

        player_card = Image.new("RGBA", (card_width, card_height), (35, 35, 35, 220))
        mask_card = Image.new("L", (card_width, card_height), 0)
        ImageDraw.Draw(mask_card).rounded_rectangle(
            (0, 0, card_width, card_height), radius=16, fill=255
        )
        player_card.putalpha(mask_card)
        image.paste(player_card, (card_x, card_y), player_card)

        draw = ImageDraw.Draw(image)

        # 3. macOS Traffic Light Dots (Left Side)[span_8](start_span)[span_8](end_span)[span_9](start_span)[span_9](end_span)
        dot_y = card_y + 18
        draw.ellipse([card_x + 20, dot_y, card_x + 32, dot_y + 12], fill=(255, 95, 86))
        draw.ellipse([card_x + 38, dot_y, card_x + 50, dot_y + 12], fill=(255, 189, 46))
        draw.ellipse([card_x + 56, dot_y, card_x + 68, dot_y + 12], fill=(40, 200, 64))

        # 4. Exact Reference Folder Icon using Image 1 (`1000177233_2.jpg`) + Text[span_10](start_span)[span_10](end_span)[span_11](start_span)[span_11](end_span)
        header_text = "Billiemusic"
        try:
            text_w = int(draw.textlength(header_text, font=self.font_header))
        except AttributeError:
            text_w = 80

        # Folder icon load karenge jo aapne bheji hai (`1000177233_2.jpg`)[span_12](start_span)[span_12](end_span)
        folder_icon_path = "1000177233_2.jpg"
        if os.path.exists(folder_icon_path):
            folder_img = Image.open(folder_icon_path).convert("RGBA")
        else:
            # Fallback agar file na ho toh blank box
            folder_img = Image.new("RGBA", (50, 50), (45, 138, 250, 255))

        icon_display_w, icon_display_h = 20, 16
        folder_resized = folder_img.resize((icon_display_w, icon_display_h), Image.Resampling.LANCZOS)

        total_header_w = icon_display_w + 6 + text_w
        start_x = int(card_x + (card_width / 2) - (total_header_w / 2))
        
        icon_y = card_y + 14
        text_y = card_y + 12

        # Paste exact folder image icon and text side-by-side seamlessly
        image.paste(folder_resized, (start_x, icon_y), folder_resized)
        draw.text((start_x + icon_display_w + 6, text_y), header_text, font=self.font_header, fill=(230, 230, 230))

        # Divider line under header
        draw.line([(card_x, card_y + 45), (card_x + card_width, card_y + 45)], fill=(55, 55, 55), width=1)

        # 5. Big Rounded Album Art Inside Window
        art_size = 380
        _art = ImageOps.fit(
            thumb, (art_size, art_size), method=Image.LANCZOS, centering=(0.5, 0.5)
        )

        mask_art = Image.new("L", (art_size, art_size), 0)
        ImageDraw.Draw(mask_art).rounded_rectangle(
            (0, 0, art_size, art_size), radius=12, fill=255
        )
        _art.putalpha(mask_art)

        art_x = card_x + 40
        art_y = card_y + 75
        image.paste(_art, (art_x, art_y), _art)

        # 6. Track Details & Typography (Right Side)
        text_x = art_x + art_size + 40
        text_y = art_y + 30

        draw.text((text_x, text_y), song.title[:30], font=self.font_title, fill=self.fill)
        draw.text((text_x, text_y + 40), song.channel_name[:30], font=self.font_sub, fill=(180, 180, 180))
        draw.text((text_x, text_y + 75), f"Views: {song.view_count}", font=self.font_sub, fill=(130, 130, 130))

        # 7. Progress Bar & Timestamps
        bar_x1 = text_x
        bar_x2 = card_x + card_width - 40
        bar_y = art_y + art_size - 40

        draw.line([(bar_x1, bar_y), (bar_x2, bar_y)], fill=(70, 70, 70), width=5, joint="curve")
        draw.line([(bar_x1, bar_y), (bar_x1 + 50, bar_y)], fill=self.fill, width=5, joint="curve")

        draw.text((bar_x1, bar_y + 10), "0:01", font=self.font_header, fill=(150, 150, 150))
        
        try:
            duration_w = int(draw.textlength(song.duration, font=self.font_header))
        except AttributeError:
            duration_w = 35

        draw.text((bar_x2 - duration_w, bar_y + 10), song.duration, font=self.font_header, fill=(150, 150, 150))

        image.save(output)
        try:
            os.remove(temp)
        except Exception:
            pass
        return output


async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    chat_id = update.effective_chat.id
    status_msg = await update.message.reply_text("🎵 Generating final macOS thumbnail...")

    try:
        track = DummyTrack()
        generator = AppleMusicThumbnailGenerator()
        output_path = await generator.generate(track)

        with open(output_path, "rb") as photo:
            await context.bot.send_photo(
                chat_id=chat_id,
                photo=photo,
                caption="✨ **Aapki exact reference wali thumbnail generate ho gayi hai!**",
                parse_mode="Markdown"
            )
        
        await status_msg.delete()
    except Exception as e:
        await status_msg.edit_text(f"❌ Error occurred: {str(e)}")


def main():
    if not BOT_TOKEN:
        print("Error: BOT_TOKEN is missing in your .env file!")
        return

    app = ApplicationBuilder().token(BOT_TOKEN).build()
    app.add_handler(CommandHandler("start", start))

    print("🤖 Bot is running... Send /start in Telegram.")
    app.run_polling()


if __name__ == "__main__":
    main()
