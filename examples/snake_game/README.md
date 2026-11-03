# BramandScript Modular Snake Game Engine

> **Production Example**: A completely modular, ANSI-rendered Snake Game featuring an autonomous heuristic AI pathfinder and turn-based interactive modes, built 100% in BramandScript (`.bmd`).

---

## 1. Modular Architecture Overview

The application demonstrates BramandScript's clean module import system, struct modeling, algorithmic pathfinding, and ANSI terminal rendering.

```
examples/snake_game/
├── point.bmd          # 2D coordinate Point struct with distance & vector arithmetic
├── direction.bmd      # Direction vectors (UP, DOWN, LEFT, RIGHT) and string parser
├── board.bmd          # Board boundaries and random food placement logic
├── snake.bmd          # Snake entity, body segment shifts, growth, and collision
├── ai_controller.bmd  # Heuristic pathfinding AI agent (wall/tail avoidance, flood freedom)
├── renderer.bmd       # ANSI color HUD and Unicode box-drawing terminal frame engine
├── game.bmd           # Core game loop coordinator for AI and interactive modes
├── snake_test.bmd     # Comprehensive unit and integration test suite
├── main.bmd           # CLI entry point and argument dispatcher
└── README.md          # Architectural documentation and guide
```

---

## 2. Running the Game

### Autonomous AI Demonstration (Autoplay)
Watch the AI agent automatically seek food, avoid walls, navigate around its growing tail, and prevent dead-ends:
```bash
bmd run examples/snake_game/main.bmd --ai
```

### Fast Demonstration Run (30 ticks)
```bash
bmd run examples/snake_game/main.bmd --demo
```

### Turn-Based Interactive Mode
Take control of the snake using standard `W/A/S/D` steering:
```bash
bmd run examples/snake_game/main.bmd --step
```
- `w` + Enter: Move Up
- `s` + Enter: Move Down
- `a` + Enter: Move Left
- `d` + Enter: Move Right
- `q` + Enter: Quit Game

### Automated Verification Test Suite
Execute the built-in test suite verifying point geometry, collision math, and AI heuristics:
```bash
bmd test examples/snake_game/snake_test.bmd
```

---

## 3. Key Technical Highlights
- **100% Pure BramandScript**: No external libraries or C extensions.
- **Unicode & ANSI Compliant**: Clean border frames (`┏━┓`, `┃`, `┗━┛`), bold snake head (`●`), body segments (`■`), and food markers (`✦`).
- **Autonomous AI Solver**: Calculates Manhattan distances to food combined with a neighborhood freedom heuristic to prevent trapped dead-ends.
- **Strict Separation of Concerns**: Pure geometry in `point.bmd`, boundary physics in `board.bmd`, entity state in `snake.bmd`, and visual representation in `renderer.bmd`.
