# BramandScript Tic-Tac-Toe Game

A high-performance, terminal-based Tic-Tac-Toe game authored 100% in BramandScript (`.bmd`).

## Features
- **Instant Single-Key Input**: Uses BramandScript's native `term_raw(true)` and `read_key()` primitives to capture cell choices (`1`-`9`) instantly without pressing Enter.
- **In-Place Screen Refresh**: Renders the game board without terminal scrolling artifacts using ANSI escape sequences (`\033[2J\033[H`).
- **Continuous Replay Loop**: Tracks cumulative scores for Player X, Player O, and Draws across multiple matches.
- **Emoji-Free Standard**: Complies with BramandScript's No-Emojis Policy using clean Unicode box-drawing typography (`┌───┬───┐`, `✦`, `│`, `─`).

## Running the Game

```bash
bmd tic_tac_toe.bmd
```
