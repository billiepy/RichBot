import os
import asyncio
import aiohttp
from dotenv import load_dotenv

load_dotenv()

BOT_TOKEN = os.getenv("BOT_TOKEN")

if not BOT_TOKEN:
    raise RuntimeError("BOT_TOKEN is missing from .env")

API = f"https://api.telegram.org/bot{BOT_TOKEN}"


async def telegram(session, method, **params):
    async with session.post(f"{API}/{method}", json=params) as response:
        data = await response.json()

        if not data.get("ok"):
            print(f"Telegram API error: {data}")
            return None

        return data["result"]


# ---------------------------------------------------------
# MAIN MENU
# ---------------------------------------------------------

def main_menu():
    return {
        "inline_keyboard": [
            [
                {
                    "text": "⌃ Chat Settings",
                    "callback_data": "settings_open",
                    "style": "primary"
                }
            ],
            [
                {
                    "text": "Help",
                    "callback_data": "help",
                    "style": "primary"
                },
                {
                    "text": "Home",
                    "callback_data": "home",
                    "style": "primary"
                }
            ],
            [
                {
                    "text": "Close",
                    "callback_data": "close",
                    "style": "danger"
                }
            ]
        ]
    }


def settings_menu():
    return {
        "inline_keyboard": [
            [
                {
                    "text": "⌃ Chat Settings",
                    "callback_data": "settings_open",
                    "style": "success"
                }
            ],
            [
                {
                    "text": "Help",
                    "callback_data": "help",
                    "style": "primary"
                },
                {
                    "text": "Home",
                    "callback_data": "home",
                    "style": "primary"
                }
            ],
            [
                {
                    "text": "Close",
                    "callback_data": "close",
                    "style": "danger"
                }
            ]
        ]
    }


# ---------------------------------------------------------
# RICH MESSAGE CONTENT
# ---------------------------------------------------------

def home_content():
    return """
<h2>Chat Owner Commands</h2>

<p>Configuration options available to the chat owner.</p>

<details>
<summary>Chat Settings</summary>

<table bordered>
<tr>
<th>Command</th>
<th>Description</th>
</tr>

<tr>
<td><code>/settings</code></td>
<td>
Manage chat settings, including play mode,
administrator mode, command auto-delete,
and language preferences.
</td>
</tr>
</table>

</details>

<img src="tg://photo?id=home_image" alt="Home Image"/>

<p><i>Use the buttons below to go back.</i></p>
"""


def settings_content():
    return """
<h2>Chat Owner Commands</h2>

<p>Configuration options available to the chat owner.</p>

<details open>
<summary>Chat Settings</summary>

<table bordered>
<tr>
<th>Command</th>
<th>Description</th>
</tr>

<tr>
<td><code>/settings</code></td>
<td>
Manage chat settings, including play mode,
administrator mode, command auto-delete,
and language preferences.
</td>
</tr>
</table>

</details>

<p><i>Use the buttons below to go back.</i></p>
"""


def help_content():
    return """
<h2>Help</h2>

<p>
Use the buttons below to navigate through the bot.
</p>

<p>
<strong>Chat Owner Commands</strong><br>
Commands available to the owner of the chat.
</p>
"""


# ---------------------------------------------------------
# SEND RICH MESSAGE
# ---------------------------------------------------------

async def send_start(session, chat_id):
    return await telegram(
        session,
        "sendRichMessage",
        chat_id=chat_id,
        rich_message={
            "html": home_content(),
            "media": [
                {
                    "id": "home_image",
                    "media": {
                        "type": "photo",
                        "media": "https://files.catbox.moe/18god1.jpg"
                    }
                }
            ]
        },
        reply_markup=main_menu()
    )

async def edit_rich_message(session, chat_id, message_id, content, keyboard):
    return await telegram(
        session,
        "editMessageText",
        chat_id=chat_id,
        message_id=message_id,
        rich_message={
            "html": content
        },
        reply_markup=keyboard
    )


# ---------------------------------------------------------
# UPDATE LOOP
# ---------------------------------------------------------

async def main():
    print("Bot started.")

    offset = 0

    timeout = aiohttp.ClientTimeout(total=40)

    async with aiohttp.ClientSession(timeout=timeout) as session:

        while True:

            try:

                updates = await telegram(
                    session,
                    "getUpdates",
                    offset=offset,
                    timeout=30,
                    allowed_updates=[
                        "message",
                        "callback_query"
                    ]
                )

                if updates is None:
                    await asyncio.sleep(2)
                    continue

                for update in updates:

                    offset = update["update_id"] + 1

                    # -------------------------------------
                    # /start
                    # -------------------------------------

                    message = update.get("message")

                    if message:

                        text = message.get("text", "")
                        chat_id = message["chat"]["id"]

                        if text.startswith("/start"):
                            await send_start(
                                session,
                                chat_id
                            )

                    # -------------------------------------
                    # CALLBACK BUTTON
                    # -------------------------------------

                    callback = update.get("callback_query")

                    if callback:

                        callback_id = callback["id"]
                        data = callback.get("data")

                        callback_message = callback.get("message")

                        if not callback_message:
                            continue

                        chat_id = callback_message["chat"]["id"]
                        message_id = callback_message["message_id"]

                        # Stop Telegram loading animation
                        await telegram(
                            session,
                            "answerCallbackQuery",
                            callback_query_id=callback_id
                        )

                        # ---------------------------------
                        # CHAT SETTINGS
                        # ---------------------------------

                        if data == "settings_open":

                            await edit_rich_message(
                                session,
                                chat_id,
                                message_id,
                                settings_content(),
                                settings_menu()
                            )

                        elif data == "settings_close":

                            await edit_rich_message(
                                session,
                                chat_id,
                                message_id,
                                home_content(),
                                main_menu()
                            )

                        # ---------------------------------
                        # HOME
                        # ---------------------------------

                        elif data == "home":

                            await edit_rich_message(
                                session,
                                chat_id,
                                message_id,
                                home_content(),
                                main_menu()
                            )

                        # ---------------------------------
                        # HELP
                        # ---------------------------------

                        elif data == "help":

                            await edit_rich_message(
                                session,
                                chat_id,
                                message_id,
                                help_content(),
                                {
                                    "inline_keyboard": [
                                        [
                                            {
                                                "text": "← Back",
                                                "callback_data": "home",
                                                "style": "primary"
                                            }
                                        ]
                                    ]
                                }
                            )

                        # ---------------------------------
                        # CLOSE
                        # ---------------------------------

                        elif data == "close":

                            await telegram(
                                session,
                                "deleteMessage",
                                chat_id=chat_id,
                                message_id=message_id
                            )

            except asyncio.CancelledError:
                raise

            except Exception as e:
                print(f"Error: {e}")
                await asyncio.sleep(3)


if __name__ == "__main__":
    asyncio.run(main())
