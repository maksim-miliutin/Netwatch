"""Draws the netwatch mark and one tile per service.

Not part of the build: run it when a service is added, then upload the tile to
the application's art assets under the service's own name.
"""

import os

from PIL import Image, ImageDraw, ImageFont

BOLD = "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"

INK = (18, 24, 30)
PAPER = (247, 246, 243)

HERE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# The tag is netwatch's own shorthand, not the service's mark: a logo belongs to
# whoever owns it, and this is a picture in somebody else's application.
SERVICES = {
    "youtube": ("yt", "#e23a2e"),
    "rutube": ("rt", "#e5197b"),
    "yandex-music": ("ym", "#e0b900"),
    "vk-video": ("vk", "#2a7cf7"),
    "twitch": ("tw", "#8b5cf6"),
    "dzen": ("dz", "#58697a"),
    "ok-video": ("ok", "#f2820c"),
    "vimeo": ("vm", "#1ab7ea"),
    "dailymotion": ("dm", "#0d6efd"),
    "coub": ("cb", "#22b573"),
    "netflix": ("nf", "#b81d24"),
    "kinopoisk": ("kp", "#ff6600"),
    "okko": ("oko", "#7a3cf0"),
    "ivi": ("ivi", "#ff005c"),
    "wink": ("wk", "#12c2a5"),
    "premier": ("pr", "#ff3b30"),
    "kick": ("kk", "#3fbf1f"),
    "vkplay": ("vp", "#14a0f2"),
    "youtube-music": ("ytm", "#ff0033"),
    "spotify": ("sp", "#1db954"),
    "soundcloud": ("sc", "#ff5500"),
    "apple-music": ("am", "#fa2d48"),
    "deezer": ("dzr", "#a238ff"),
    "zvuk": ("zv", "#00d1a0"),
    "mixcloud": ("mx", "#5000ff"),
    "tiktok": ("tt", "#25f4ee"),
    "bilibili": ("bl", "#00a1d6"),
    "crunchyroll": ("cr", "#f47521"),
    "disney-plus": ("d+", "#0c3fa5"),
    "max": ("max", "#002be7"),
    "prime-video": ("pv", "#00a8e1"),
    "apple-tv": ("atv", "#4a4a4a"),
    "hulu": ("hu", "#1ce783"),
    "nebula": ("nb", "#26355a"),
    "odysee": ("od", "#ef1970"),
    "nicovideo": ("nn", "#3a3a3a"),
    "trovo": ("tr", "#19d66c"),
}


def rgb(colour):
    return tuple(int(colour[i:i + 2], 16) for i in (1, 3, 5))


def mixed(colour, towards, amount):
    return tuple(round(c * (1 - amount) + t * amount) for c, t in zip(colour, towards))


def light(colour):
    red, green, blue = colour

    return (0.299 * red + 0.587 * green + 0.114 * blue) / 255


def tile(tag, colour, size=1024):
    """One service: its colour, its tag, a border set in from the edge."""
    paint = rgb(colour)
    big = size * 4

    image = Image.new("RGB", (big, big), paint)
    draw = ImageDraw.Draw(image)

    edge = round(big * 0.045)
    draw.rectangle(
        [edge, edge, big - edge - 1, big - edge - 1],
        outline=mixed(paint, INK, 0.30),
        width=round(big * 0.012),
    )

    # A dark tag on a pale tile and a pale one on a dark tile. Contrast decides
    # this, not taste: at the size Discord draws a card, the tag is all there is.
    ink = mixed(paint, INK, 0.80) if light(paint) > 0.45 else mixed(paint, PAPER, 0.85)

    font = ImageFont.truetype(BOLD, round(big * (0.46 if len(tag) < 3 else 0.32)))
    left, top, right, bottom = draw.textbbox((0, 0), tag, font=font)

    draw.text(
        ((big - (right - left)) / 2 - left, (big - (bottom - top)) / 2 - top),
        tag,
        font=font,
        fill=ink,
    )

    return image.resize((size, size), Image.LANCZOS)


def mark(size=1024, gone=130):
    """The netwatch mark: a disc with a wedge taken out. How much has gone."""
    big = size * 4

    image = Image.new("RGB", (big, big), INK)
    draw = ImageDraw.Draw(image)

    pad = big * 0.135
    box = [pad, pad, big - pad, big - pad]

    # A wedge starts at twelve o'clock, as on a clock, and not at three, as in
    # PIL. Hence the minus ninety.
    draw.pieslice(box, -90 + gone, -90 + 360, fill=PAPER)

    hole = big * 0.05
    draw.pieslice(
        [box[0] + hole, box[1] + hole, box[2] - hole, box[3] - hole],
        -90,
        -90 + gone,
        fill=INK,
    )

    return image.resize((size, size), Image.LANCZOS)


def main():
    art = os.path.join(HERE, "art")
    icons = os.path.join(HERE, "extension", "icons")

    os.makedirs(art, exist_ok=True)
    os.makedirs(icons, exist_ok=True)

    for name, (tag, colour) in SERVICES.items():
        tile(tag, colour).save(os.path.join(art, name + ".png"))

    mark().save(os.path.join(art, "netwatch.png"))

    for size in (16, 32, 48, 128):
        mark(size).save(os.path.join(icons, str(size) + ".png"))

    print("drawn", len(SERVICES), "tiles")


if __name__ == "__main__":
    main()
