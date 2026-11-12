"""Rebuild the app's brand assets from logo.png and banner.png in this folder.

    python -I make_assets.py

Writes, relative to the app folder:
    frontend/src/assets/logo.png    the logo's ink as transparency (the page paints it in the theme's colour)
    frontend/src/assets/mark.png    the X alone, the same way
    frontend/src/assets/banner.jpg  the banner, sized for the page
    build/appicon.png               the window and .exe icon (Wails makes icon.ico from it)
"""
from pathlib import Path

from PIL import Image, ImageChops, ImageDraw, ImageOps

HERE = Path(__file__).resolve().parent
APP = HERE.parent
ASSETS = APP / "frontend" / "src" / "assets"

SLANT = 0.36   # the logo's italic slope, dx/dy
X_END = 580    # where the X ends once the slope is taken out


def ink(logo: Image.Image):
    """The logo's black shapes as an alpha channel, cropped to them."""
    logo = logo.convert("RGBA")
    dark = ImageChops.multiply(ImageOps.invert(logo.convert("L")), logo.getchannel("A"))
    # Drop the faint halo the white outline leaves; keep the smooth edges.
    dark = dark.point(lambda v: 0 if v < 70 else 255 if v > 200 else int((v - 70) * 255 / 130))
    return dark.crop(dark.point(lambda v: 255 if v > 128 else 0).getbbox())


def as_mask(alpha: Image.Image):
    out = Image.new("RGBA", alpha.size, (0, 0, 0, 0))
    out.putalpha(alpha)
    return out


def x_glyph(alpha: Image.Image):
    """The X, cut from the wordmark along the letters' own slope."""
    w, h = alpha.size
    upright = alpha.transform((w + int(SLANT * h), h), Image.AFFINE, (1, -SLANT, 0, 0, 1, 0), resample=Image.BICUBIC)
    upright = upright.crop((0, 0, X_END, h))
    slanted = upright.transform((X_END, h), Image.AFFINE, (1, SLANT, 0, 0, 1, 0), resample=Image.BICUBIC)
    return slanted.crop(slanted.point(lambda v: 255 if v > 128 else 0).getbbox())


def fit(alpha: Image.Image, width: int):
    return alpha.resize((width, round(alpha.size[1] * width / alpha.size[0])), Image.LANCZOS)


def main():
    ASSETS.mkdir(parents=True, exist_ok=True)
    alpha = ink(Image.open(HERE / "logo.png"))
    as_mask(fit(alpha, 1240)).save(ASSETS / "logo.png", optimize=True)

    x = x_glyph(alpha)
    as_mask(fit(x, 320)).save(ASSETS / "mark.png", optimize=True)

    size = 1024
    icon = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    ImageDraw.Draw(icon).rounded_rectangle((0, 0, size - 1, size - 1), radius=220, fill=(10, 10, 10, 255))
    glyph = fit(x, 760)
    white = Image.new("RGBA", glyph.size, (255, 255, 255, 255))
    icon.paste(white, ((size - glyph.size[0]) // 2, (size - glyph.size[1]) // 2), glyph)
    (APP / "build").mkdir(exist_ok=True)
    icon.save(APP / "build" / "appicon.png", optimize=True)

    banner = Image.open(HERE / "banner.png").convert("RGB")
    banner = banner.resize((1600, round(banner.size[1] * 1600 / banner.size[0])), Image.LANCZOS)
    banner.save(ASSETS / "banner.jpg", quality=84, optimize=True, progressive=True)

    for p in (ASSETS / "logo.png", ASSETS / "mark.png", ASSETS / "banner.jpg", APP / "build" / "appicon.png"):
        print(f"{p.relative_to(APP)}  {Image.open(p).size}  {p.stat().st_size // 1024} KB")


if __name__ == "__main__":
    main()
