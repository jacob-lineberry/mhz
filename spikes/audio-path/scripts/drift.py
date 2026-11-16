"""S1 Q3: estimate clock drift from a cmd/play CSV.

    python3 -I scripts/drift.py run.csv [skip_seconds]

Fits straight lines (least squares) from skip_seconds on:
  arrival   = decoded frames vs wall time  -> server's sample clock
  consume   = frames oto pulled vs wall time -> sound card's sample clock
  buffered  = decoded - pulled              -> the gap between the two
Rates are in frames per wall-clock second; ppm is relative to 44100.
"""
import csv
import sys


def fit(xs, ys):
    n = len(xs)
    mx, my = sum(xs) / n, sum(ys) / n
    sxx = sum((x - mx) ** 2 for x in xs)
    sxy = sum((x - mx) * (y - my) for x, y in zip(xs, ys))
    b = sxy / sxx
    resid = [y - (my + b * (x - mx)) for x, y in zip(xs, ys)]
    return b, max(abs(r) for r in resid)


rows = list(csv.DictReader(open(sys.argv[1])))
skip = float(sys.argv[2]) if len(sys.argv) > 2 else 120
rows = [r for r in rows if float(r["t_s"]) >= skip]
t = [float(r["t_s"]) for r in rows]
dec = [int(r["dec_frames"]) for r in rows]
pl = [int(r["played_frames"]) for r in rows]
buf = [d - p for d, p in zip(dec, pl)]
nom = 44100
a, ar = fit(t, dec)
c, cr = fit(t, pl)
b, br = fit(t, buf)
span = t[-1] - t[0]
print(f"{sys.argv[1]}: span {span:.0f}s from t={t[0]:.0f}s")
print(f"  arrival  {a:10.2f} fr/s  ({(a / nom - 1) * 1e6:+7.1f} ppm vs system clock), max residual {ar / nom * 1000:.0f} ms")
print(f"  consume  {c:10.2f} fr/s  ({(c / nom - 1) * 1e6:+7.1f} ppm vs system clock), max residual {cr / nom * 1000:.0f} ms")
print(f"  buffered slope {b / nom * 3600 * 1000:+8.1f} ms/hour ({b / nom * 1e6:+.1f} ppm), start {buf[0] / nom:.2f}s end {buf[-1] / nom:.2f}s, jitter +-{br / nom * 1000:.0f} ms")
lv = [float(r["ring_ms"]) + float(r["player_ms"]) for r in rows]
lb, lr = fit(t, lv)
print(f"  LEVEL slope {lb * 3600:+8.1f} ms/hour ({lb * 1000:+.1f} ppm), min {min(lv) / 1000:.2f}s max {max(lv) / 1000:.2f}s, jitter +-{lr:.0f} ms")
starves = rows[-1]["starves"]
print(f"  starves={starves} starved_ms={rows[-1]['starved_ms']} gaps={rows[-1]['gaps']} gap_ms={rows[-1]['gap_ms']} dropped={rows[-1]['dropped']}")
cpu = (float(rows[-1]["cpu_ms"]) - float(rows[0]["cpu_ms"])) / 1000 / span * 100
print(f"  cpu {cpu:.2f}% of one core, heap {min(float(r['heap_mb']) for r in rows):.1f}-{max(float(r['heap_mb']) for r in rows):.1f} MB")
