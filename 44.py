import os
from pyrogram import Client, filters
from pyrogram.types import Message, InlineKeyboardMarkup, InlineKeyboardButton
from dotenv import load_dotenv

# .env file se variables load karein
load_dotenv()

BOT_TOKEN = os.getenv("BOT_TOKEN")
API_ID = os.getenv("API_ID")
API_HASH = os.getenv("API_HASH")

# Pyrogram Client initialize karein
app = Client(
    "rich_bot",
    api_id=API_ID,
    api_hash=API_HASH,
    bot_token=BOT_TOKEN
)

# Buttons layout (Screenshot ke jaisa)
start_keyboard = InlineKeyboardMarkup(
    [
        [
            InlineKeyboardButton("AFK", callback_data="afk"),
            InlineKeyboardButton("Admin", callback_data="admin"),
            InlineKeyboardButton("Autoarchive", callback_data="autoarchive"),
        ],
        [
            InlineKeyboardButton("Billiard", callback_data="billiard"),
            InlineKeyboardButton("Blacklist", callback_data="blacklist"),
        ],
        [
            InlineKeyboardButton("ChatBot", callback_data="chatbot"),
            InlineKeyboardButton("CodeTester", callback_data="codetester"),
            InlineKeyboardButton("Currency", callback_data="currency"),
        ],
        [
            InlineKeyboardButton("DevCommand", callback_data="devcommand"),
            InlineKeyboardButton("Federation", callback_data="federation"),
            InlineKeyboardButton("Filters", callback_data="filters"),
        ],
        [
            InlineKeyboardButton("◀", callback_data="prev"),
            InlineKeyboardButton("Back", callback_data="back"),
            InlineKeyboardButton("▶", callback_data="next"),
        ]
    ]
)

@app.on_message(filters.command("start"))
async def start_handler(client: Client, message: Message):
    # User ka naam le kar text banayein
    user_name = message.from_user.first_name if message.from_user else "User"
    text = (
        f"Hello 🐾 {user_name}, My name is MissKaty 🐯.\n"
        "I'm a bot with some useful features.\n"
        "You can change language bot using /setlang command, it's still in beta stage.\n\n"
        "You can choose an option below, by clicking a button."
    )
    
    # Message bhejein with inline buttons
    sent_msg = await message.reply(text, reply_markup=start_keyboard)
    
    # Khud ke message par big aur animated fire reaction dein
    try:
        await client.send_reaction(
            chat_id=message.chat.id,
            message_id=sent_msg.id,
            emoji="🔥",
            big=True  # Yeh reaction ko animated aur bada/floating effect dega
        )
    except Exception as e:
        print(f"Reaction error: {e}")

if __name__ == "__main__":
    print("Bot start ho raha hai...")
    app.run()
