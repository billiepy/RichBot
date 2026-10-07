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
DEFAULT_IMAGE_URL = os.getenv("IMAGE", "https://picsum.photos/1280/720") # Catbox ki jagah highly stable placeholder image link use ki hai

os.makedirs("cache", exist_ok=True)


class DummyTrack:
    def __init__(self):
        self.id = "sample_track_123"
        self.title = "Anuv Jain - Hoshiyar (Apple Music Vibe)"
        self.channel_name = "Anuv Jain Official"
        self.view_count = "1.2M"
        self.duration = "3:45"
        self.thumbnail = DEFAULT_IMAGE_URL


class AppleMusicThumbnailGenerator:

    def __init__(self):
        self.fill = (255, 255, 255)
        try:
            self.font_title = ImageFont.truetype(
                "anony/helpers/Raleway-Bold.ttf", 28
            )
            self.font_sub = ImageFont.truetype(
                "anony/helpers/Inter-Light.ttf", 22
            )
            self.font_time = ImageFont.truetype(
                "anony/helpers/Inter-Light.ttf", 18
            )
        except Exception:
            self.font_title = ImageFont.load_default()
            self.font_sub = ImageFont.load_default()
            self.font_time = ImageFont.load_default()

    def save_thumb(self, output_path: str, url: str) -> str:
        headers = {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
        }
        
        # Retry mechanism taaki connection drop hone par session auto-reconnect kare
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
        except Exception as e:
            # Agar online download fail ho jaye, toh ek solid fallback color ki image generate kar dega taaki bot crash na ho
            img = Image.new("RGB", (1280, 720), (30, 30, 30))
            img.save(output_path)
            
        return output_path

    async def generate(self, song: DummyTrack, size=(1280, 720)) -> str:
        temp = f"cache/temp_{song.id}.jpg"
        output = f"cache/{song.id}.png"

        self.save_thumb(temp, song.thumbnail)

        # 1. Background: Blurred & Darkened Album Art
        thumb = Image.open(temp).convert("RGBA").resize(size, Image.Resampling.LANCZOS)
        blur = thumb.filter(ImageFilter.GaussianBlur(40))
        image = ImageEnhance.Brightness(blur).enhance(0.30)

        # 2. Central Glassmorphism Player Card (Apple Music Style)
        card_width, card_height = 960, 520
        card_x = (size[0] - card_width) // 2
        card_y = (size[1] - card_height) // 2

        player_card = Image.new("RGBA", (card_width, card_height), (20, 20, 20, 160))
        mask_card = Image.new("L", (card_width, card_height), 0)
        ImageDraw.Draw(mask_card).rounded_rectangle(
            (0, 0, card_width, card_height), radius=24, fill=255
        )
        player_card.putalpha(mask_card)
        image.paste(player_card, (card_x, card_y), player_card)

        # 3. Big Rounded Album Art
        art_size = 400
        _art = ImageOps.fit(
            thumb, (art_size, art_size), method=Image.LANCZOS, centering=(0.5, 0.5)
        )

        mask_art = Image.new("L", (art_size, art_size), 0)
        ImageDraw.Draw(mask_art).rounded_rectangle(
            (0, 0, art_size, art_size), radius=16, fill=255
        )
        _art.putalpha(mask_art)

        art_x = card_x + 60
        art_y = card_y + 60
        image.paste(_art, (art_x, art_y), _art)

        # 4. Track Details & Typography
        draw = ImageDraw.Draw(image)
        text_x = art_x + art_size + 50
        text_y = art_y + 40

        draw.text((text_x, text_y), song.title[:32], font=self.font_title, fill=self.fill)
        draw.text((text_x, text_y + 45), song.channel_name[:30], font=self.font_sub, fill=(200, 200, 200))
        draw.text((text_x, text_y + 80), f"{song.view_count} views", font=self.font_sub, fill=(140, 140, 140))

        # 5. Progress Bar & Controls Area
        bar_x1 = text_x
        bar_x2 = card_x + card_width - 60
        bar_y = art_y + art_size - 60

        draw.line([(bar_x1, bar_y), (bar_x2, bar_y)], fill=(70, 70, 70), width=6, joint="curve")
        draw.line([(bar_x1, bar_y), (bar_x1 + 60, bar_y)], fill=self.fill, width=6, joint="curve")

        draw.text((bar_x1, bar_y + 12), "0:01", font=self.font_time, fill=(160, 160, 160))
        
        try:
            duration_w = draw.textlength(song.duration, font=self.font_time)
        except AttributeError:
            duration_w = 40

        draw.text((bar_x2 - duration_w, bar_y + 12), song.duration, font=self.font_time, fill=(160, 160, 160))

        image.save(output)
        try:
            os.remove(temp)
        except Exception:
            pass
        return output


async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    chat_id = update.effective_chat.id
    status_msg = await update.message.reply_text("🎵 Generating Apple Music thumbnail...")

    try:
        track = DummyTrack()
        generator = AppleMusicThumbnailGenerator()
        output_path = await generator.generate(track)

        with open(output_path, "rb") as photo:
            await context.bot.send_photo(
                chat_id=chat_id,
                photo=photo,
                caption="✨ **Apple Music Thumbnail generated successfully!**",
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
