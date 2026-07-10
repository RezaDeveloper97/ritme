---
name: swipe-back
description: Add or verify the mandatory Telegram-style edge swipe-to-go-back gesture on a screen. Use whenever creating a new screen, building navigation, or touching the SwipeBackContainer. Every non-root screen must support it. Hand-built, no library.
---

# Telegram-style Swipe-to-Go-Back

CLAUDE.md §5b. Every non-root screen is dismissible by dragging from the left edge
toward the right: the screen follows the finger 1:1, the previous screen shows
underneath with subtle parallax/dim, release past threshold pops, release before it
snaps back. No gesture/navigation library — Compose primitives only (§3).

## Implementation

- One reusable `SwipeBackContainer` composable wraps each screen's content and calls
  the nav `pop` on commit. Do NOT re-implement per screen.
- Detect drags with `pointerInput` + `detectHorizontalDragGestures` (or `AnchoredDraggable`).
- Track the horizontal offset with an `Animatable`; apply it via `Modifier.offset`/
  custom drawing — do not recompose a heavy tree each frame (§5 perf).
- Show the previous screen beneath during the drag (parallax + dim), like Telegram.
- **Edge-start only:** begin tracking from the left ~20dp zone so it doesn't fight
  inner horizontal scrolling / `LazyRow`.
- **Commit threshold:** pop at >~40% width dragged or a fast fling; otherwise spring
  back to 0 with a quick, natural spring.

## Checklist

- [ ] Screen is wrapped in the shared `SwipeBackContainer` (root/start screen excluded).
- [ ] Drag follows the finger in real time, no lag; previous screen visible underneath.
- [ ] Commit threshold + fling handled; snap-back animates when below threshold.
- [ ] Gesture starts only from the left edge zone; inner horizontal scroll still works.
- [ ] System Back button/gesture performs the exact same `pop` (parity).
- [ ] No third-party gesture/navigation library added.
