"""Gem Hunter teaser video builder.

Renders a 1920x1080 @30fps kinetic-typography teaser (5 scenes from
.memory/teaser.md), pipes raw frames to ffmpeg, muxes teaser.wav VO.

  python teaser_build.py

Requires: Pillow, numpy. ffmpeg: CapCut build (mpeg4 + aac).
Output: teaser.mp4 (add to .gitignore - do NOT commit binaries).
"""
import math
import subprocess
import sys
import wave

from PIL import Image, ImageDraw, ImageFont

W, H, FPS = 1920, 1080, 30
FFMPEG = r"C:\Users\RED DEMON\AppData\Local\CapCut\Apps\1.4.0.198\ffmpeg.exe"
AUDIO = "teaser.wav"
OUT = "teaser.mp4"

# Palette cloned from web/static/css/app.css
BG = (12, 10, 9)
CARD = (28, 25, 23)
BORDER = (41, 37, 36)
TEXT = (242, 242, 242)
MUTED = (161, 161, 170)
CRIMSON = (158, 1, 66)
CRIMSON_LT = (213, 62, 79)
GREEN = (171, 221, 164)
AMBER = (253, 174, 97)
TEAL = (102, 194, 165)
BLACK = (0, 0, 0)

FONT = r"C:\Windows\Fonts\arial.ttf"
FONTB = r"C:\Windows\Fonts\arialbd.ttf"


def font(size, bold=False):
    try:
        return ImageFont.truetype(FONTB if bold else FONT, size)
    except OSError:
        return ImageFont.load_default()


F_TITLE = font(132, True)
F_BIG = font(104, True)
F_MED = font(58, True)
F_BODY = font(46)
F_SMALL = font(38)
F_TAPE = font(44, True)
F_CHIP = font(32, True)


def audio_duration(path):
    with wave.open(path) as w:
        return w.getnframes() / w.getframerate()


DUR = audio_duration(AUDIO)
NFRAMES = math.ceil(DUR * FPS)
print(f"audio {AUDIO}: {DUR:.2f}s -> {NFRAMES} frames @ {FPS}fps")

# Scene boundaries: storyboard seconds (of 55) scaled to real VO duration.
STORY = [("HOOK", 0, 8), ("PROBLEM", 8, 18), ("SOLUTION", 18, 28),
         ("RISK CHECK", 28, 38),          ("AUTOMATION", 38, 50), ("OUTRO", 50, 55)]
SCENES = [(name, a / 55 * DUR, b / 55 * DUR) for name, a, b in STORY]
for name, a, b in SCENES:
    print(f"  {name:10s} {a:5.2f}s - {b:5.2f}s")


def clamp01(x):
    return max(0.0, min(1.0, x))


def ease_out(x):
    x = clamp01(x)
    return 1 - (1 - x) ** 3


# Static vertical-gradient background, built once.
def make_bg():
    import numpy as np
    grad = np.linspace(0, 1, H)[:, None]
    top = np.array([22, 16, 18], dtype=float)
    bot = np.array([10, 8, 8], dtype=float)
    arr = (top + (bot - top) * grad).astype("uint8")
    arr = np.repeat(arr[:, None, :], W, axis=1)
    return Image.fromarray(arr, "RGB")


BG_IMG = make_bg()

TICKERS = [("BBCA", +1.2), ("BBRI", -0.6), ("TLKM", +0.8), ("ASII", +2.1),
           ("BRIS", +3.4), ("ANTM", -1.8), ("HRUM", +1.1), ("ESSA", -0.4),
           ("ITMG", +2.6), ("ADRO", +0.5), ("BMRI", -0.9), ("INDF", +1.7)]


def tape_lane(d, y, t, speed, flip=False):
    items = []
    for tk, pc in TICKERS:
        col = GREEN if pc >= 0 else CRIMSON_LT
        s = f"  {tk} {'+' if pc >= 0 else ''}{pc:.1f}%  "
        items.append((s, col))
    widths = [d.textlength(s, font=F_TAPE) for s, _ in items]
    gap = 40
    lane = sum(widths) + gap * len(widths)
    off = (t * speed) % lane
    if flip:
        off = lane - off
    x = -off
    for _ in range(2):  # draw twice to cover wrap
        for (s, col), wdt in zip(items, widths):
            if -wdt < x < W:
                d.text((x, y), s, font=F_TAPE, fill=col)
            x += wdt + gap
        x -= lane  # second pass starts one lane back


def text_c(d, y, s, fnt, fill):
    bb = d.textbbox((0, 0), s, font=fnt)
    d.text(((W - (bb[2] - bb[0])) / 2 - bb[0], y), s, font=fnt, fill=fill)


def card(d, x0, y0, x1, y1, outline=BORDER):
    d.rounded_rectangle([x0, y0, x1, y1], radius=18, fill=CARD, outline=outline, width=2)


def chip(d, cx, y, s, fill):
    bb = d.textbbox((0, 0), s, font=F_CHIP)
    w = bb[2] - bb[0] + 56
    d.rounded_rectangle([cx - w / 2, y, cx + w / 2, y + 58], radius=29,
                        outline=fill, width=2)
    d.text((cx - (bb[2] - bb[0]) / 2 - bb[0], y + 10), s, font=F_CHIP, fill=fill)


def header(d, scene_name, t):
    d.rectangle([0, 0, W, 6], fill=CRIMSON)
    s = f"GEM HUNTER  ·  {scene_name}"
    d.text((48, 36), s, font=F_CHIP, fill=MUTED)
    pw = W * clamp01(t / DUR)
    d.rectangle([0, H - 8, pw, H], fill=CRIMSON)


def scene_hook(img, d, lt, dur, t):
    tape_lane(d, 150, t, 260)
    tape_lane(d, H - 210, t, 200, flip=True)
    k1 = ease_out(lt / (dur * 0.35))
    d.text((960 - 560 * (1 - k1) - 480, 400), "900+ IDX STOCKS",
           font=F_TITLE, fill=TEXT, anchor="lm")
    if lt > dur * 0.32:
        k2 = ease_out((lt - dur * 0.32) / (dur * 0.25))
        pop = 1 + 0.25 * (1 - k2)
        fnt = font(int(104 * pop), True)
        bb = d.textbbox((0, 0), "→ TOP 10?", font=fnt)
        d.text(((W - (bb[2] - bb[0])) / 2 - bb[0], 560), "→ TOP 10?",
               font=fnt, fill=AMBER)
    if lt > dur * 0.62:
        text_c(d, 760, "Which 10 deserve your attention today?", F_BODY, MUTED)


def scene_problem(img, d, lt, dur, t):
    if lt < dur * 0.32:
        text_c(d, 430, "DATA IS EVERYWHERE.", F_BIG, TEXT)
    else:
        k = ease_out((lt - dur * 0.30) / (dur * 0.22))
        x0, y0, x1, y1 = 560, 300, 1360, 560
        card(d, x0, y0, x1, y1)
        lines = ['{ "price": 8120, "eps": null,', '  "bvps": "??", "pe": -3.2 }',
                 '=SCREENER_v7_FINAL(2).xlsx', 'note: "cek manual ya..."']
        for i, ln in enumerate(lines):
            if lt > dur * (0.30 + 0.06 * i):
                d.text((x0 + 40, y0 + 36 + i * 52), ln, font=F_SMALL, fill=MUTED)
        # red X draws progressively
        cx, cy = (x0 + x1) / 2, (y0 + y1) / 2
        L = 260 * k
        d.line([cx - L, cy - L * 0.55, cx + L, cy + L * 0.55], fill=CRIMSON_LT, width=22)
        if k > 0.55:
            k2 = (k - 0.55) / 0.45
            d.line([cx + L * k2, cy - L * 0.55 * k2, cx - L * k2, cy + L * 0.55 * k2],
                   fill=CRIMSON_LT, width=22)
    if lt > dur * 0.58:
        k3 = ease_out((lt - dur * 0.58) / (dur * 0.2))
        text_c(d, 660, "CHEAP ≠ VALUE", font(int(104 * (1 + 0.2 * (1 - k3))), True),
               CRIMSON_LT)
    if lt > dur * 0.78:
        text_c(d, 830, "— or a value trap.", F_BODY, MUTED)


def scene_solution(img, d, lt, dur, t):
    text_c(d, 200, "MEET GEM HUNTER.", F_BIG, TEXT)
    if lt > dur * 0.15:
        chip(d, 960, 360, "POWERED BY SECTORS.APP", TEAL)
    cards = [("GRAHAM · VALUE", "√(22.5·EPS·BVPS)"),
             ("LYNCH · GROWTH", "PEG = PE/growth"),
             ("GL SCORE", "0.55·G + 0.45·L")]
    cw, chh, gap = 460, 260, 40
    x0 = (W - (3 * cw + 2 * gap)) / 2
    for i, (title, formula) in enumerate(cards):
        st = dur * (0.32 + 0.14 * i)
        if lt > st:
            k = ease_out((lt - st) / (dur * 0.18))
            y = 560 + 220 * (1 - k)
            cx = x0 + i * (cw + gap) + cw / 2
            card(d, x0 + i * (cw + gap), y, x0 + i * (cw + gap) + cw, y + chh,
                 outline=CRIMSON if i == 2 else BORDER)
            d.text((cx, y + 70), title, font=F_CHIP, anchor="mm",
                   fill=AMBER if i == 2 else MUTED)
            d.text((cx, y + 160), formula, font=F_SMALL, anchor="mm", fill=TEXT)


def scene_risk(img, d, lt, dur, t):
    text_c(d, 170, "THEN WE CHECK THE RISK.", F_BIG, TEXT)
    rows = [("GEM GUARD", "flags pump-risk · HIGH badge", CRIMSON_LT),
            ("GEM SENTINEL", "flags distress · Critical zone", AMBER),
            ("AI EXPLAINED", "guardrailed analysis · no hype", GREEN)]
    for i, (name, desc, col) in enumerate(rows):
        st = dur * (0.18 + 0.20 * i)
        if lt > st:
            k = ease_out((lt - st) / (dur * 0.16))
            y = 360 + i * 200
            x = 260 - 500 * (1 - k)
            card(d, x, y, x + 1400, y + 150)
            d.rectangle([x, y + 20, x + 12, y + 130], fill=col)
            d.text((x + 48, y + 28), name, font=F_MED, fill=col)
            d.text((x + 48, y + 92), desc, font=F_SMALL, fill=MUTED)


def scene_auto(img, d, lt, dur, t):
    text_c(d, 150, "RUNS EVERY 6 HOURS.", F_BIG, TEXT)
    x0, y0, x1, y1 = 360, 330, 1560, 800
    card(d, x0, y0, x1, y1)
    logs = [("$ scheduler tick — universe sync (Sectors.app)", TEAL),
            ("$ rank run — 128 eligible → Top 10 set", TEXT),
            ("$ worker — AI analysis done (10/10)", GREEN),
            ("$ mail — digest sent · curl /api/v1/ranking/top10 → 200", AMBER)]
    for i, (ln, col) in enumerate(logs):
        if lt > dur * (0.22 + 0.13 * i):
            d.text((x0 + 44, y0 + 40 + i * 62), ln, font=F_SMALL, fill=col)
    if lt > dur * 0.78:
        chip(d, 760, 860, "EMAIL DIGEST", GREEN)
        chip(d, 1180, 860, "{} JSON API", TEAL)


def scene_close(img, d, lt, dur, t):
    text_c(d, 300, "GEM HUNTER", F_TITLE, TEXT)
    d.rectangle([660, 480, 1260, 492], fill=CRIMSON)
    if lt > dur * 0.2:
        text_c(d, 540, "Find the signal. Check the risk.", F_MED, AMBER)
    if lt > dur * 0.4:
        chip(d, 960, 680, "gemhunter.afjn.site", TEAL)
    text_c(d, 800, "Research tool. Not financial advice.", F_SMALL, MUTED)
    text_c(d, 880, "Sectors Hackathon 2026 · Market Intelligence", F_SMALL, MUTED)


RENDER = {"HOOK": scene_hook, "PROBLEM": scene_problem, "SOLUTION": scene_solution,
          "RISK CHECK": scene_risk,           "AUTOMATION": scene_auto, "OUTRO": scene_close}


def frame_at(t):
    img = BG_IMG.copy()
    d = ImageDraw.Draw(img)
    name = SCENES[-1][0]
    lt = dur = 1
    for n, a, b in SCENES:
        if a <= t < b or (t >= b and n == SCENES[-1][0]):
            name, lt, dur = n, t - a, b - a
    RENDER[name](img, d, lt, dur, t)
    header(d, name, t)
    if t < 0.4:  # fade in
        img = Image.blend(Image.new("RGB", (W, H), BLACK), img, t / 0.4)
    if t > DUR - 0.5:  # fade out
        img = Image.blend(img, Image.new("RGB", (W, H), BLACK),
                          (t - (DUR - 0.5)) / 0.5)
    return img


def main():
    cmd = [FFMPEG, "-y",
           "-f", "rawvideo", "-pix_fmt", "rgb24", "-s", f"{W}x{H}",
           "-r", str(FPS), "-i", "-",
           "-i", AUDIO,
           "-c:v", "mpeg4", "-q:v", "2",
           "-c:a", "aac", "-b:a", "128k",
           "-shortest", "-movflags", "+faststart", OUT]
    try:
        proc = subprocess.Popen(cmd, stdin=subprocess.PIPE,
                                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    except OSError as e:
        print(f"cannot start ffmpeg: {e}")
        sys.exit(1)
    for f in range(NFRAMES):
        img = frame_at(f / FPS)
        try:
            proc.stdin.write(img.tobytes())
        except BrokenPipeError:
            print("ffmpeg pipe broke")
            sys.exit(1)
        if f % 150 == 0:
            print(f"  frame {f}/{NFRAMES} ({f / NFRAMES * 100:.0f}%)")
    proc.stdin.close()
    rc = proc.wait()
    print(f"ffmpeg exit {rc} -> {OUT}" if rc == 0 else f"ffmpeg FAILED ({rc})")
    sys.exit(rc)


if __name__ == "__main__":
    main()
