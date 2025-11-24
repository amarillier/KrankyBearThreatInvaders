# KrankyBear ThreatInvaders

A cross-platform Space Invaders-style game built with Go and the Fyne GUI library. Defend against CVE and KB threats (invaders) while earning points and progressing through increasingly difficult levels.

## Features

### Gameplay
- **Classic Space Invaders gameplay** - Defend against waves of invaders
- **CVE and KB Threats** - Two types of invaders with different point values:
  - CVEs (red squares/diamonds): Worth 100 points
  - KBs (blue circles/rounded): Worth 50 points
- **Bullet Neutralization** - Your bullets can neutralize enemy bullets for 5 bonus points
- **Progressive Difficulty** - Game gets harder as you progress through levels
- **High Score Tracking** - Track your best scores locally

### Controls
- **Arrow Keys**: Move defender left/right
- **Space Bar**: Fire bullets at invaders
- **Mouse**: 
  - Move mouse left/right to control defender
  - Click to fire bullets
- **P**: Pause/Resume the game
- **F12**: Boss key - pause the game and hide the window (quickly hide the game)
- **Z**: Cheat key - add one life (deducts 100 points per life)

### Difficulty Modes
- **Normal**: 3 lives (default)
- **Medium**: 5 lives
- **Easy**: 8 lives

Difficulty can be changed from the menu or system tray (only when not playing).

### Sound Effects
- **wheeHoo.mp3**: Plays when completing a level
- **zip.mp3** and **boing.mp3**: Alternate when hitting invaders
- **explosion.mp3**: Plays when losing a life
- Sound effects can be toggled on/off via:
  - Side panel button
  - Main menu
  - System tray menu

### Platform Support
- **Cross-Platform**: Works on macOS, Windows, and Linux
  - **Linux**: Works on all desktop environments (GNOME, KDE, XFCE, Cinnamon, MATE, etc.) with X11 or Wayland
  - **macOS**: macOS 10.13 (High Sierra) or later
  - **Windows**: Windows 10 or later

### UI Features
- **System Tray Integration**: 
  - Show/Hide window
  - Difficulty selection
  - Sound effects toggle
  - About, Help, Update Check, and Quit options
- **Window Management**: 
  - All dialogs (About, Help, Update Check, Reset Scores) open on the same display as the game window
  - Window tracking ensures only one instance of each dialog can be open
- **Update Checker**: Built-in update checking with links to latest version and release notes
- **Custom Icons**: All windows display the KrankyBear USArmy Combat Helmet Woodland icon
- **Tanium Logo Support**: When launched with "tanium" in the executable name (case-insensitive), the defender sprite uses the Tanium logo instead of the default shield
- **Dynamic App Name**: App name changes based on executable name:
  - "KrankyBear Tanium Threat Invaders" if "tanium" is in the executable name
  - "KrankyBear ThreatInvaders" otherwise

### Technical Features
- **Single Executable**: All resources (images, sounds) are embedded in the binary
- **No External Dependencies**: Resources directory not required at runtime
- **Preferences Storage**: Game settings (difficulty, sound) are saved and persist between sessions

## Dependencies

- [Fyne](https://fyne.io/) v2.6.3 - Cross-platform GUI toolkit
- [gopxl/beep](https://github.com/gopxl/beep) - Audio playback library
- [go-update-checker](https://github.com/amarillier/go-update-checker) - Update checking functionality
- Go standard library

## Building

See the build scripts in the repository:
- `compile-mac.sh` - Build macOS binaries
- `compile-linux.sh` - Build Linux binaries
- `compile-windows.ps1` - Build Windows binaries
- `package.sh` - Create installers (.pkg for macOS, .deb/.rpm for Linux)
- `compile-all.sh` - Build and package for all platforms

## Installation

### macOS
Install the `.pkg` installer from the `installers/` directory.

### Linux
Install the `.deb` (Debian/Ubuntu) or `.rpm` (RedHat/Fedora) package from the `installers/` directory.

### Windows
Run the installer from the `installers/` directory.

## License

This project is provided as-is, free for personal, educational and commercial use, under GNU GPL-3.0

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## Author

Allan Marillier

## Acknowledgments

- Built with [Fyne](https://fyne.io/) - An easy-to-use GUI toolkit for Go
- Inspired by the classic Space Invaders arcade game
