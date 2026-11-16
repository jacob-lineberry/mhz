# Faceplate visual brief

The look of the car-stereo faceplate. Changes 6, 7, 8 and 10 follow it; ask before departing
from it. Researched 2026-11-16. `<angle brackets>` mark decisions still open. The order of
work is in [docs/roadmap.md](../roadmap.md).

**Mood.** A car stereo at night: a black panel, one glowing display, a few lit legends.
Calm when idle, alive when music plays. It is a device, so it always paints its own dark
background and ignores the terminal's light or dark theme.

**Palette.** No Lospec palette has VFD cyan, amber and magenta on a true near-black: the
closest, [AAP-64][lospec-aap64], has a magenta at 4.4:1 on its darkest color, and the
others have navy or purple "blacks" or lack one of the three hues. So this is a custom
palette of 11 roles. Contrast is WCAG against the background, computed. The hexes
approximate the common VFD phosphor (peak [505 nm][vfd]); they are untested on your
screens, so judge them in the first stills.

| Role | Hex | Contrast | Used for |
|---|---|---|---|
| Background | `#07090c` | — | Everything behind the panel |
| Panel | `#10151b` | 1.1 | Faceplate body |
| Bezel | `#2a323c` | 1.5 | Borders, dial ticks |
| Ghost | `#0f3a36` | 1.6 | Unlit segments and empty analyzer cells |
| VFD dim | `#1c8f86` | 5.1 | Secondary readouts, scale labels |
| VFD | `#3df2e0` | 14.2 | Frequency, station, title, bars |
| VFD hot | `#b8fff6` | 17.7 | Momentary emphasis: a digit just changed |
| Amber | `#ffb000` | 10.9 | Things you set: volume, active preset, stored state |
| Amber dim | `#9a5a00` | 3.6 | Inactive presets |
| Magenta | `#ff3ea5` | 6.2 | The accent: needle, peak caps, the live dot |
| Dim text | `#6f7b88` | 4.6 | Key hints, credits |

**Color rules.**

- Cyan means "from the stream", amber means "set by you", magenta marks at most three
  things on screen.
- State is never carried by color alone. At 16 colors the dim steps collapse, so each state
  also has a glyph or word (`● LIVE`, `MUTE`, `[3]`).
- The palette lives in one place as named roles. Themes can come later as data.
- Design for 256 colors as the floor: Terminal.app is truecolor only "since macOS 26"
  ([termstandard]). Windows Terminal, iTerm2, Ghostty and WezTerm are truecolor.
- Charm's `colorprofile` v0.4.3 parses `NO_COLOR` as a boolean, so `NO_COLOR=1` works but
  an arbitrary non-empty value does not (read in source). Test `NO_COLOR=1`.

**Typography in a terminal.** Three levels, set by size, then brightness:

1. The frequency, in three-row digits built from heavy box-drawing characters. It is the
   only large element.
2. Station name in capitals at VFD brightness; the track title in normal case.
3. Everything else in dim text or VFD dim.

Use only block elements, box drawing and plain ASCII. Avoid emoji, anything
double-width, and the "Symbols for Legacy Computing" block (segmented digits, sextants),
which [often renders as boxes][legacy-computing]. Bold only on the station name.

**Layout at 80x24** (a sketch; the analyzer is empty until M4):

```text
╭─ mhz ────────────────────────────────────────────────────────────────────────╮
│                                                                              │
│  FM  ┏━┓ ┏━┓   ╺━┓        NEON DRIVE                               ST  ▂▄▆█  │
│      ┣━┫ ┣━┫     ┃  MHz   synthwave · outrun                       320k MP3  │
│      ┗━┛ ┗━┛ ▪   ╹        via Nightride FM                           ● LIVE  │
│                                                                              │
│        ▼                                                                     │
│  88 ╶────┬────┬────┬────┬────┬────┬────┬────┬────┬────╴ 108                  │
│                                                                              │
│              ▃▃▃ ▇▇▇ ▁▁▁         ▄▄▄                                         │
│          ▆▆▆ ███ ███ ███ ▃▃▃ ▇▇▇ ███ ▅▅▅     ▁▁▁                             │
│  ▁▁▁ ▆▆▆ ███ ███ ███ ███ ███ ███ ███ ███ ▇▇▇ ███ ▄▄▄ ▁▁▁ ▃▃▃                 │
│  ███ ███ ███ ███ ███ ███ ███ ███ ███ ███ ███ ███ ███ ███ ███ ▆▆▆ ▄▄▄ ▂▂▂     │
│  63          250             1k              4k              16k             │
│                                                                              │
│  NOW  Artist Name - Track Title                                              │
│                                                                              │
│  [1] 88.7   [2] 91.3   [3] 94.1   [4] 97.9   [5] 101.5   [6] 105.3           │
│                                                                              │
│  VOL ████████████░░░░░░░░  60                                                │
│                                                                              │
│                                                                              │
│  ←/→ tune  1-6 preset  ↑/↓ vol  m mute  space play/stop  ? help  q quit      │
╰──────────────────────────────────────────────────────────────────────────────╯
```

| Size | Behavior |
|---|---|
| 80x24 | The faceplate fills the window, as above. Analyzer 4 rows |
| 120x40 | The faceplate grows to about 100x30 and is centered. Analyzer 8–10 rows, more bands, five-row digits. The margin is background |
| 240x70 | The faceplate stops at about 120x34 and is centered; it never stretches. Around it: <plain background | the scene, after M5> |
| Under 60 columns or 16 rows | Compact strip: frequency, station, title, key hints. No dial, analyzer or presets |
| Under 40 columns or 6 rows | One line: `88.7 NEON DRIVE  Artist - Title` |
| Smaller | Clipped to the window, never a panic (the existing shell requirement) |

**Restraint, and what to borrow.**

| Practice | Seen in | Use here |
|---|---|---|
| Key-hint footer with short and full modes | [bubbles] `help`; [k9s] `?` | Footer plus `?` overlay (change 7) |
| Focus shown by border color alone | [lazygit][lazygit-config] `activeBorderColor` | Active preset is amber; nothing else changes |
| Rounded borders by default | lazygit `border: rounded` | Rounded outer bezel, no inner boxes |
| A clear "too small" fallback | lazygit's minimum-size message | The compact tiers above |
| Layout that adapts to the window | lazygit `portraitMode: auto` | Size tiers, not stretching |
| Themes as data | k9s skins, lazygit theme config | Palette as named roles from day one |
| One accent on gray | [gum] examples | Magenta, three uses at most |

What makes those tools feel finished is consistency more than decoration: the same
spacing everywhere, one focus signal, hints always in the same place, and nothing that
moves without a reason. Hold the faceplate to that before adding motion.

Unsourced: details of specific 1980s Pioneer, Kenwood or Alpine faceplates. The elements
used here (frequency, band, ST indicator, presets 1–6, analyzer with fast attack and slow
decay) are generic to the era ([Audio, Dec 1980][audio-1980]).

## Sources

Checked 2026-11-16.

[lospec-aap64]: https://lospec.com/palette-list/aap-64
[vfd]: https://en.wikipedia.org/wiki/Vacuum_fluorescent_display
[termstandard]: https://github.com/termstandard/colors
[legacy-computing]: https://en.wikipedia.org/wiki/Symbols_for_Legacy_Computing
[bubbles]: https://github.com/charmbracelet/bubbles
[k9s]: https://github.com/derailed/k9s
[lazygit-config]: https://github.com/jesseduffield/lazygit/blob/master/docs/Config.md
[gum]: https://github.com/charmbracelet/gum
[audio-1980]: https://gammaelectronics.xyz/audio_12-1980_meters.html
