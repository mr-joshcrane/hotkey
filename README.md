# Keystroke Trainer

Practice StarCraft: Brood War hotkey patterns until they become muscle memory.

## Usage

```
go build -o keystroketrainer.exe
./keystroketrainer.exe
```

## Controls

- **SPACE/ENTER** - Start session
- **ESC** - Stop session
- **Click anywhere** - Focus window

## Pattern Format

Patterns are loaded from `keystroke_patterns.txt` (same directory as executable, or working directory).

```
# Comments start with #
Pattern Name|pattern
```

### Tokens

| Token | Input |
|-------|-------|
| `a-z`, `0-9` | Keyboard keys |
| `F1`-`F12` | Function keys |
| `LC` | Left click |
| `RC` | Right click |
| `MC` | Middle click |
| `SLC` | Shift + Left click |
| `SRC` | Shift + Right click |

### Examples

```
# Cycle through 3 control groups, attack-move each
3 Army Cycle|1aLC2aLC3aLC

# Siege tanks on group 1 and 4
Tank Siege|1z4z

# Irradiate spell cloning: cast, shift-click to queue, repeat
Irradiate Clone|5cLCSLCcLCSLCcLCSLCcLCSLC

# Alternate between F4 and F3 locations with clicks
Rally Cycle|F4LCF3RCF4LCF3RCF4LCF3RC

# Queue production across multiple factories
SixFactoryAllIn|3w4w5q6q7q8q
```

## How It Works

1. Patterns shuffle each session
2. Type the pattern exactly as shown
3. Mistakes reset your progress on that pattern
4. Failed patterns repeat later in the session
5. Session ends when all patterns are completed without mistakes

Stats are saved to `keystroke_stats.json`.
