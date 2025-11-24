package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	updatechecker "github.com/amarillier/go-update-checker"
)

const (
	GameCanvasWidth  = GameWidth
	GameCanvasHeight = GameHeight
)

// ShortcutBossKey describes the boss key shortcut (F12) - pause and hide window
// Using F12 as it's commonly used for boss keys and less likely to conflict
type ShortcutBossKey struct{}

var _ fyne.KeyboardShortcut = (*ShortcutBossKey)(nil)

// Key returns the KeyName for this shortcut
func (s *ShortcutBossKey) Key() fyne.KeyName {
	return fyne.KeyF12
}

// Mod returns the KeyModifier for this shortcut
func (s *ShortcutBossKey) Mod() fyne.KeyModifier {
	return fyne.KeyModifierShortcutDefault
}

// ShortcutName returns the shortcut name
func (s *ShortcutBossKey) ShortcutName() string {
	return "BossKey"
}

// GameUI manages the game's user interface
type GameUI struct {
	game             *Game
	app              fyne.App
	window           fyne.Window
	gameCanvas       *canvas.Raster
	scoreLabel       *widget.Label
	livesLabel       *widget.Label
	levelLabel       *widget.Label
	statusLabel      *widget.Label
	highScoreLabel   *widget.Label
	startButton      *widget.Button
	pauseButton      *widget.Button
	resetScoreButton *widget.Button
	soundButton      *widget.Button
	highScoreMgr     *HighScoreManager
	cveMgr           *CVEManager
	keysPressed      map[fyne.KeyName]bool
	canvasWidth      int
	canvasHeight     int
	mouseX           float64     // Track mouse X position for movement
	resetDialog      fyne.Window // Track reset confirmation dialog
	aboutDialog      fyne.Window // Track about dialog window
	helpDialog       fyne.Window // Track help dialog window
	updateDialog     fyne.Window // Track update check dialog window
	showMenuItem     *fyne.MenuItem
	hideMenuItem     *fyne.MenuItem
	windowVisible    bool           // Track window visibility state
	versionStatus    string         // Track version status: "current", "newer", or "unknown"
	difficultyMode   DifficultyMode // Current difficulty mode
	normalMenuItem   *fyne.MenuItem
	mediumMenuItem   *fyne.MenuItem
	easyMenuItem     *fyne.MenuItem
	taniumImage      image.Image    // Cached Tanium logo image
	useTaniumLogo    bool           // Whether to use Tanium logo based on executable name
	soundManager     *SoundManager  // Sound effects manager
	soundMenuItem    *fyne.MenuItem // Menu item to toggle sound
	prevLives        int            // Track previous lives count for explosion sound
	prevLevel        int            // Track previous level for wheeHoo sound
	prevScore        int            // Track previous score for invader hit sounds
	invaderHitCount  int            // Counter to alternate between zip and boing
}

// NewGameUI creates a new game UI
func NewGameUI(app fyne.App) *GameUI {
	cveMgr := NewCVEManager()
	cveData, _ := cveMgr.FetchCVEData()
	kbData, _ := cveMgr.FetchKBData()

	// Load difficulty from preferences (default to Normal)
	prefs := app.Preferences()
	difficultyInt := prefs.IntWithFallback("difficulty_mode", int(DifficultyNormal))
	difficultyMode := DifficultyMode(difficultyInt)
	if difficultyMode < DifficultyNormal || difficultyMode > DifficultyEasy {
		difficultyMode = DifficultyNormal
	}

	// Load sound preference (default to enabled)
	soundEnabled := prefs.BoolWithFallback("sound_enabled", true)

	game := NewGame(cveData, kbData, difficultyMode)

	// Initialize sound manager
	soundMgr, err := NewSoundManager(soundEnabled)
	if err != nil {
		// If sound initialization fails, continue without sound
		soundMgr = nil
	}

	// Detect executable name to determine if we should use Tanium logo
	execPath := os.Args[0]
	execName := filepath.Base(execPath)
	// Remove extension if present (e.g., .exe on Windows)
	execNameWithoutExt := strings.TrimSuffix(execName, filepath.Ext(execName))
	useTaniumLogo := strings.Contains(strings.ToLower(execNameWithoutExt), "tanium")

	// Decode Tanium logo image (only if we'll use it)
	var taniumImg image.Image
	if useTaniumLogo {
		if imgData, _, err := image.Decode(bytes.NewReader(resourceTaniumPngData)); err == nil {
			taniumImg = imgData
		} else {
			// Fallback: create a simple placeholder if decoding fails
			taniumImg = image.NewRGBA(image.Rect(0, 0, 1, 1))
		}
	}

	ui := &GameUI{
		game:            game,
		app:             app,
		window:          app.NewWindow(appName),
		highScoreMgr:    NewHighScoreManager(app),
		cveMgr:          cveMgr,
		keysPressed:     make(map[fyne.KeyName]bool),
		canvasWidth:     GameCanvasWidth,
		canvasHeight:    GameCanvasHeight,
		mouseX:          -1,   // -1 means mouse not over canvas
		windowVisible:   true, // Window starts visible
		versionStatus:   "unknown",
		difficultyMode:  difficultyMode,
		taniumImage:     taniumImg,
		useTaniumLogo:   useTaniumLogo,
		soundManager:    soundMgr,
		prevLives:       difficultyMode.GetLives(),
		prevLevel:       1,
		prevScore:       0,
		invaderHitCount: 0,
	}

	// Set window icon
	ui.window.SetIcon(resourceKrankyBearUSArmyCombatHelmetWoodlandPng)

	ui.setupUI()
	ui.setupKeyboard()
	ui.setupMenu()

	// Check for updates on startup
	ui.checkForUpdates()

	// Setup system tray after everything else is ready
	ui.setupSystemTray()

	return ui
}

// setupUI sets up the user interface
func (ui *GameUI) setupUI() {
	// Create game canvas - let it fill available space
	ui.gameCanvas = canvas.NewRaster(ui.drawGame)
	// Don't set fixed size - let it expand to fill space

	// Create labels
	ui.scoreLabel = widget.NewLabel("Score: 0")
	ui.livesLabel = widget.NewLabel("Lives: 3")
	ui.levelLabel = widget.NewLabel("Level: 1")
	ui.statusLabel = widget.NewLabel("Press Start to begin")
	ui.statusLabel.Alignment = fyne.TextAlignCenter
	ui.highScoreLabel = widget.NewLabel("High Scores:\n" + ui.formatHighScores())

	// Create buttons
	ui.startButton = widget.NewButton("Start", ui.onStart)
	ui.pauseButton = widget.NewButton("Pause", ui.onPause)
	ui.pauseButton.Disable()
	ui.resetScoreButton = widget.NewButton("Reset Scores", ui.onResetScores)

	// Sound effects toggle button
	ui.soundButton = widget.NewButton("Sound: On", func() {
		ui.toggleSound()
		ui.updateSoundButtonText()
	})
	ui.updateSoundButtonText()

	// Layout
	infoPanel := container.NewVBox(
		ui.scoreLabel,
		ui.livesLabel,
		ui.levelLabel,
		ui.statusLabel,
		widget.NewSeparator(),
		ui.highScoreLabel,
		widget.NewSeparator(),
		ui.startButton,
		ui.pauseButton,
		ui.resetScoreButton,
		ui.soundButton,
	)

	// Wrap canvas in mouse-enabled widget
	mouseCanvas := newMouseCanvas(ui.gameCanvas, ui)

	// Layout: game on left, controls on right (like Tetris)
	content := container.NewBorder(
		nil, nil, nil, infoPanel,
		mouseCanvas,
	)

	ui.window.SetContent(content)
	// Make window larger and allow resizing
	ui.window.Resize(fyne.NewSize(1000, 700))
	ui.window.CenterOnScreen()

	// Handle window close button - quit the application
	ui.window.SetCloseIntercept(func() {
		ui.app.Quit()
	})

	// Track canvas size changes
	ui.gameCanvas.Resize(fyne.NewSize(800, 600))

	// Start game loop
	go ui.gameLoop()
}

// mouseCanvas is a custom widget that wraps the raster canvas and handles mouse events
type mouseCanvas struct {
	widget.BaseWidget
	raster *canvas.Raster
	ui     *GameUI
}

var _ desktop.Mouseable = (*mouseCanvas)(nil)

func newMouseCanvas(raster *canvas.Raster, ui *GameUI) *mouseCanvas {
	mc := &mouseCanvas{
		raster: raster,
		ui:     ui,
	}
	mc.ExtendBaseWidget(mc)
	return mc
}

func (mc *mouseCanvas) CreateRenderer() fyne.WidgetRenderer {
	return &mouseCanvasRenderer{
		objects: []fyne.CanvasObject{mc.raster},
		mc:      mc,
	}
}

func (mc *mouseCanvas) MouseMoved(ev *desktop.MouseEvent) {
	// Always track mouse position, not just when playing (for smoother transitions)
	// ev.Position is relative to the widget
	size := mc.Size()

	// Get mouse position relative to widget
	// ev.Position is already relative to the widget, but we need to ensure it's within bounds
	relX := ev.Position.X

	// Update canvas width for scaling calculations (use actual widget size)
	// This is critical for proper mouse-to-game coordinate conversion
	if int(size.Width) > 0 {
		mc.ui.canvasWidth = int(size.Width)
	}

	// Update mouse position if within widget bounds
	// Track the full width including the right edge
	if relX >= 0 && relX <= size.Width {
		mc.ui.mouseX = float64(relX)
	} else if relX < 0 {
		mc.ui.mouseX = 0
	} else if relX > size.Width {
		mc.ui.mouseX = float64(size.Width)
	} else {
		mc.ui.mouseX = -1
	}
}

func (mc *mouseCanvas) MouseIn(ev *desktop.MouseEvent) {
	// Mouse entered the widget - start tracking
	mc.MouseMoved(ev)
}

func (mc *mouseCanvas) MouseOut() {
	// Mouse left the widget - stop tracking
	mc.ui.mouseX = -1
}

func (mc *mouseCanvas) MouseDown(ev *desktop.MouseEvent) {
	if mc.ui.game.State == StatePlaying {
		mc.ui.game.Shoot()
	}
}

func (mc *mouseCanvas) MouseUp(ev *desktop.MouseEvent) {
	// Not needed but required by interface
}

func (mc *mouseCanvas) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

type mouseCanvasRenderer struct {
	objects []fyne.CanvasObject
	mc      *mouseCanvas
}

func (r *mouseCanvasRenderer) Layout(size fyne.Size) {
	r.mc.raster.Resize(size)
	// Update canvas width for mouse tracking
	if r.mc.ui != nil {
		r.mc.ui.canvasWidth = int(size.Width)
	}
}

func (r *mouseCanvasRenderer) MinSize() fyne.Size {
	return r.mc.raster.MinSize()
}

func (r *mouseCanvasRenderer) Refresh() {
	r.mc.raster.Refresh()
}

func (r *mouseCanvasRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *mouseCanvasRenderer) Destroy() {}

// setupKeyboard sets up keyboard and mouse controls
func (ui *GameUI) setupKeyboard() {
	// Add boss key shortcut (F12) - pause game and hide window
	// Using F12 as it's commonly used for boss keys and less likely to conflict
	ui.window.Canvas().AddShortcut(&ShortcutBossKey{}, func(shortcut fyne.Shortcut) {
		ui.bossKey()
	})

	// Use typed key events for better responsiveness
	ui.window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		// Check for boss key (F12)
		if ev.Name == fyne.KeyF12 {
			ui.bossKey()
			return
		}

		if ev.Name == fyne.KeySpace {
			ui.game.Shoot()
		} else if ev.Name == fyne.KeyP {
			ui.game.Pause()
		} else if ev.Name == fyne.KeyZ {
			// Cheat key: add a life
			if ui.game.State == StatePlaying || ui.game.State == StatePaused {
				ui.game.AddLife()
				ui.updateLabels()
			}
		} else if ev.Name == fyne.KeyLeft {
			ui.keysPressed[fyne.KeyLeft] = true
			if ui.game.State == StatePlaying {
				ui.game.MovePlayer(-1)
			}
		} else if ev.Name == fyne.KeyRight {
			ui.keysPressed[fyne.KeyRight] = true
			if ui.game.State == StatePlaying {
				ui.game.MovePlayer(1)
			}
		}
	})
}

// bossKey handles the boss key (F12) - pauses the game and hides the window
func (ui *GameUI) bossKey() {
	// Pause the game if playing
	if ui.game.State == StatePlaying {
		ui.game.Pause()
	}
	// Hide the window
	ui.hideWindow()
}

// setupMenu sets up the application menu
func (ui *GameUI) setupMenu() {
	showMenu := fyne.NewMenuItem("Show", func() {
		fyne.Do(ui.showWindow)
	})
	hideMenu := fyne.NewMenuItem("Hide", func() {
		fyne.Do(ui.hideWindow)
	})
	aboutMenu := fyne.NewMenuItem("About", func() {
		fyne.Do(ui.showAboutDialog)
	})
	helpMenu := fyne.NewMenuItem("Help", func() {
		fyne.Do(ui.showHelpDialog)
	})
	updateMenu := fyne.NewMenuItem("Check for Update", func() {
		ui.showUpdateDialog()
	})

	// Difficulty mode menu items
	ui.normalMenuItem = fyne.NewMenuItem("Normal (3 lives)", func() {
		ui.setDifficulty(DifficultyNormal)
	})
	ui.mediumMenuItem = fyne.NewMenuItem("Medium (5 lives)", func() {
		ui.setDifficulty(DifficultyMedium)
	})
	ui.easyMenuItem = fyne.NewMenuItem("Easy (8 lives)", func() {
		ui.setDifficulty(DifficultyEasy)
	})

	// Set initial checked state
	ui.updateDifficultyMenuState()

	difficultySubMenu := fyne.NewMenu("Difficulty",
		ui.normalMenuItem,
		ui.mediumMenuItem,
		ui.easyMenuItem,
	)
	difficultyMenuItem := fyne.NewMenuItem("Difficulty", nil)
	difficultyMenuItem.ChildMenu = difficultySubMenu

	// Sound toggle menu item
	ui.soundMenuItem = fyne.NewMenuItem("Sound Effects", func() {
		ui.toggleSound()
	})
	ui.updateSoundMenuState()

	mainMenu := fyne.NewMenu("KrankyBear ThreatInvaders",
		showMenu,
		hideMenu,
		fyne.NewMenuItemSeparator(),
		difficultyMenuItem,
		ui.soundMenuItem,
		fyne.NewMenuItemSeparator(),
		aboutMenu,
		helpMenu,
		updateMenu,
	)

	menu := fyne.NewMainMenu(mainMenu)
	ui.window.SetMainMenu(menu)
}

// setDifficulty sets the difficulty mode and restarts the game if playing
func (ui *GameUI) setDifficulty(difficulty DifficultyMode) {
	ui.difficultyMode = difficulty
	ui.game.Difficulty = difficulty

	// Save to preferences
	prefs := ui.app.Preferences()
	prefs.SetInt("difficulty_mode", int(difficulty))

	// Update menu state
	ui.updateDifficultyMenuState()

	// If game is in menu or game over, update lives display
	if ui.game.State == StateMenu || ui.game.State == StateGameOver {
		ui.game.Lives = difficulty.GetLives()
		ui.updateLabels()
	} else if ui.game.State == StatePlaying || ui.game.State == StatePaused {
		// If playing, restart with new difficulty
		ui.game.Start()
		ui.updateLabels()
	}
}

// updateDifficultyMenuState updates the checkmarks on difficulty menu items
func (ui *GameUI) updateDifficultyMenuState() {
	ui.normalMenuItem.Checked = (ui.difficultyMode == DifficultyNormal)
	ui.mediumMenuItem.Checked = (ui.difficultyMode == DifficultyMedium)
	ui.easyMenuItem.Checked = (ui.difficultyMode == DifficultyEasy)
}

// toggleSound toggles sound effects on/off
func (ui *GameUI) toggleSound() {
	if ui.soundManager == nil {
		return
	}
	enabled := !ui.soundManager.IsEnabled()
	ui.soundManager.SetEnabled(enabled)

	// Save to preferences
	prefs := ui.app.Preferences()
	prefs.SetBool("sound_enabled", enabled)

	// Update menu state
	ui.updateSoundMenuState()
	// Update button text
	ui.updateSoundButtonText()
	// Update system tray menu
	ui.updateTrayMenuState()
}

// updateSoundMenuState updates the sound menu item checkmark
func (ui *GameUI) updateSoundMenuState() {
	if ui.soundMenuItem != nil && ui.soundManager != nil {
		ui.soundMenuItem.Checked = ui.soundManager.IsEnabled()
	}
}

// updateSoundButtonText updates the sound button text based on enabled state
func (ui *GameUI) updateSoundButtonText() {
	if ui.soundButton != nil && ui.soundManager != nil {
		if ui.soundManager.IsEnabled() {
			ui.soundButton.SetText("Sound: On")
		} else {
			ui.soundButton.SetText("Sound: Off")
		}
	}
}

// checkForUpdates checks for updates and updates version status
func (ui *GameUI) checkForUpdates() {
	go func() {
		uc := updatechecker.New("amarillier", "KrankyBearThreatInvaders", "KrankyBear ThreatInvaders", "https://github.com/amarillier/KrankyBearThreatInvaders/releases/latest", 0, false)
		uc.CheckForUpdate(appVersion)

		fyne.Do(func() {
			// Check if we're running a newer version than released
			if strings.Contains(uc.Message, "running a newer version") || strings.Contains(uc.Message, "newer than") {
				ui.versionStatus = "newer" // We're running newer than released
			} else if strings.Contains(uc.Message, "running the latest") || strings.Contains(uc.Message, "up to date") {
				ui.versionStatus = "current" // Version matches released
			} else if uc.UpdateAvailable {
				ui.versionStatus = "update" // Update available
			} else {
				ui.versionStatus = "unknown"
			}
		})
	}()
}

// drawGame draws the game
func (ui *GameUI) drawGame(width, height int) image.Image {
	// Update canvas height for scaling (width is updated by mouse canvas widget)
	ui.canvasHeight = height
	// Don't update canvasWidth here - it's managed by the mouse canvas widget
	// to ensure it matches the actual widget size for mouse tracking

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Calculate scaling factors
	scaleX := float64(width) / float64(GameWidth)
	scaleY := float64(height) / float64(GameHeight)

	// Draw background (dark space)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{10, 10, 20, 255})
		}
	}

	// Draw stars (background effect)
	ui.drawStars(img, width, height)

	if ui.game.State == StatePlaying || ui.game.State == StatePaused {
		// Draw invaders (scaled)
		for _, row := range ui.game.Invaders {
			for _, invader := range row {
				if invader != nil && !invader.Destroyed {
					ui.drawInvaderScaled(img, invader, scaleX, scaleY)
				}
			}
		}

		// Draw player bullets (scaled)
		for _, bullet := range ui.game.PlayerBullets {
			if bullet.Active {
				ui.drawBulletScaled(img, bullet, true, scaleX, scaleY)
			}
		}

		// Draw invader bullets (scaled)
		for _, bullet := range ui.game.InvaderBullets {
			if bullet.Active {
				ui.drawBulletScaled(img, bullet, false, scaleX, scaleY)
			}
		}

		// Draw player (scaled)
		ui.drawPlayerScaled(img, ui.game.Player, scaleX, scaleY)
	}

	// Draw pause overlay (semi-transparent so game remains visible)
	if ui.game.State == StatePaused {
		ui.drawOverlay(img, width, height, "PAUSED", color.RGBA{0, 0, 0, 128})
	}

	// Draw game over overlay
	if ui.game.State == StateGameOver {
		ui.drawOverlay(img, width, height, "GAME OVER", color.RGBA{255, 0, 0, 200})
	}

	// Draw level complete overlay
	if ui.game.State == StateLevelComplete {
		ui.drawOverlay(img, width, height, "LEVEL COMPLETE!", color.RGBA{0, 255, 0, 200})
	}

	return img
}

// drawInvaderScaled draws an invader scaled to canvas size
func (ui *GameUI) drawInvaderScaled(img *image.RGBA, invader *Invader, scaleX, scaleY float64) {
	x := int(invader.X * scaleX)
	y := int(invader.Y * scaleY)
	w := int(invader.Width * scaleX)
	h := int(invader.Height * scaleY)

	c := color.RGBA{invader.Color[0], invader.Color[1], invader.Color[2], 255}

	switch invader.Shape {
	case "square":
		// Square shape for CVEs (red)
		ui.drawFilledRect(img, x, y, w, h, c)
		// Draw border
		ui.drawRectBorder(img, x, y, w, h, color.RGBA{255, 255, 255, 255})

	case "circle":
		// Circle shape for KBs (blue)
		ui.drawFilledCircle(img, x+w/2, y+h/2, w/2-2, c)
		// Draw border
		ui.drawCircleBorder(img, x+w/2, y+h/2, w/2-2, color.RGBA{255, 255, 255, 255})

	case "diamond":
		// Diamond shape for some CVEs
		ui.drawFilledDiamond(img, x+w/2, y+h/2, w/2-2, h/2-2, c)
		// Draw border
		ui.drawDiamondBorder(img, x+w/2, y+h/2, w/2-2, h/2-2, color.RGBA{255, 255, 255, 255})

	case "rounded":
		// Rounded square for some KBs
		ui.drawFilledRoundedRect(img, x, y, w, h, 8, c)
		// Draw border
		ui.drawRoundedRectBorder(img, x, y, w, h, 8, color.RGBA{255, 255, 255, 255})
	}

	// Draw text (CVE ID or KB number) - simplified for now
	// In a full implementation, you'd use text rendering
	ui.drawTextCentered(img, x+w/2, y+h/2, invader.Text, color.RGBA{255, 255, 255, 255})
}

// drawPlayerScaled draws the player scaled to canvas size
// Uses Tanium logo if executable name contains "tanium", otherwise uses original shield sprite
func (ui *GameUI) drawPlayerScaled(img *image.RGBA, player *Player, scaleX, scaleY float64) {
	x := int(player.X * scaleX)
	y := int(player.Y * scaleY)
	w := int(player.Width * scaleX)
	h := int(player.Height * scaleY)

	if ui.useTaniumLogo && ui.taniumImage != nil {
		// Draw Tanium logo image scaled to player size
		srcBounds := ui.taniumImage.Bounds()
		srcW := srcBounds.Dx()
		srcH := srcBounds.Dy()

		// Create a temporary RGBA image for the scaled Tanium logo
		scaledTanium := image.NewRGBA(image.Rect(0, 0, w, h))

		// Scale the image using nearest neighbor (simple but fast)
		for dy := 0; dy < h; dy++ {
			for dx := 0; dx < w; dx++ {
				// Map destination coordinates to source coordinates
				srcX := (dx * srcW) / w
				srcY := (dy * srcH) / h

				// Clamp to source bounds
				if srcX >= 0 && srcX < srcW && srcY >= 0 && srcY < srcH {
					srcColor := ui.taniumImage.At(srcBounds.Min.X+srcX, srcBounds.Min.Y+srcY)
					scaledTanium.Set(dx, dy, srcColor)
				}
			}
		}

		// Draw the scaled image onto the game canvas
		draw.Draw(img, image.Rect(x, y, x+w, y+h), scaledTanium, image.Point{0, 0}, draw.Over)
	} else {
		// Draw original shield sprite (rounded top, pointed bottom)
		playerColor := color.RGBA{0, 180, 90, 255} // Tanium green
		centerX := x + w/2
		centerY := y + h/2

		// Top rounded part (semi-circle)
		topRadius := w / 3
		for dy := 0; dy < h/2; dy++ {
			for dx := -w / 2; dx < w/2; dx++ {
				px := centerX + dx
				py := y + dy
				// Check if point is in rounded top
				if dx*dx+(dy-topRadius)*(dy-topRadius) <= topRadius*topRadius {
					if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
						img.Set(px, py, playerColor)
					}
				}
			}
		}

		// Bottom pointed part (triangle)
		bottomY := y + h/2
		for dy := 0; dy < h/2; dy++ {
			widthAtY := w * (h/2 - dy) / (h / 2)
			for dx := -widthAtY / 2; dx < widthAtY/2; dx++ {
				px := centerX + dx
				py := bottomY + dy
				if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
					img.Set(px, py, playerColor)
				}
			}
		}

		// Draw border
		ui.drawShieldBorder(img, centerX, y, w, h, color.RGBA{255, 255, 255, 255})

		// Draw "T" for Tanium in center
		ui.drawTextCentered(img, centerX, centerY, "T", color.RGBA{255, 255, 255, 255})
	}
}

// drawShieldBorder draws a border around the shield shape
func (ui *GameUI) drawShieldBorder(img *image.RGBA, centerX, topY, w, h int, c color.RGBA) {
	// Draw top arc border
	topRadius := w / 3
	for angle := 0; angle <= 180; angle++ {
		rad := float64(angle) * math.Pi / 180.0
		x := int(float64(centerX) + float64(topRadius)*math.Cos(rad))
		y := int(float64(topY) + float64(topRadius) + float64(topRadius)*math.Sin(rad))
		if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
			img.Set(x, y, c)
		}
	}

	// Draw bottom triangle borders
	bottomY := topY + h/2
	leftX := centerX - w/2
	rightX := centerX + w/2
	bottomX := centerX
	bottomYEnd := topY + h

	ui.drawLine(img, leftX, bottomY, bottomX, bottomYEnd, c)
	ui.drawLine(img, rightX, bottomY, bottomX, bottomYEnd, c)
}

// drawBulletScaled draws a bullet scaled to canvas size
func (ui *GameUI) drawBulletScaled(img *image.RGBA, bullet *Bullet, isPlayer bool, scaleX, scaleY float64) {
	x := int(bullet.X * scaleX)
	y := int(bullet.Y * scaleY)
	w := int(bullet.Width * scaleX)
	h := int(bullet.Height * scaleY)

	var c color.RGBA
	if isPlayer {
		c = color.RGBA{0, 255, 255, 255} // Cyan for player bullets
	} else {
		c = color.RGBA{255, 100, 100, 255} // Red for invader bullets
	}

	ui.drawFilledRect(img, x, y, w, h, c)
}

// drawStars draws background stars
func (ui *GameUI) drawStars(img *image.RGBA, width, height int) {
	starColor := color.RGBA{255, 255, 255, 150}
	for i := 0; i < 50; i++ {
		x := (i * 37) % width
		y := (i * 73) % height
		img.Set(x, y, starColor)
		if x+1 < width {
			img.Set(x+1, y, starColor)
		}
		if y+1 < height {
			img.Set(x, y+1, starColor)
		}
	}
}

// drawOverlay draws a semi-transparent overlay with text
func (ui *GameUI) drawOverlay(img *image.RGBA, width, height int, text string, bgColor color.RGBA) {
	// Draw semi-transparent background by blending with existing pixels
	alpha := float64(bgColor.A) / 255.0
	invAlpha := 1.0 - alpha

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Get existing pixel color
			existing := img.At(x, y)
			r, g, b, _ := existing.RGBA()
			existingR := uint8(r >> 8)
			existingG := uint8(g >> 8)
			existingB := uint8(b >> 8)

			// Blend colors
			newR := uint8(float64(existingR)*invAlpha + float64(bgColor.R)*alpha)
			newG := uint8(float64(existingG)*invAlpha + float64(bgColor.G)*alpha)
			newB := uint8(float64(existingB)*invAlpha + float64(bgColor.B)*alpha)

			img.Set(x, y, color.RGBA{newR, newG, newB, 255})
		}
	}

	// Draw text in center (simplified - just draw a rectangle to indicate text)
	textX := width / 2
	textY := height / 2
	textColor := color.RGBA{255, 255, 255, 255}
	ui.drawTextCentered(img, textX, textY, text, textColor)
}

// Helper drawing functions

func (ui *GameUI) drawFilledRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			if x+dx < img.Bounds().Dx() && y+dy < img.Bounds().Dy() && x+dx >= 0 && y+dy >= 0 {
				img.Set(x+dx, y+dy, c)
			}
		}
	}
}

func (ui *GameUI) drawRectBorder(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	// Top and bottom
	for dx := 0; dx < w; dx++ {
		if x+dx < img.Bounds().Dx() {
			if y >= 0 && y < img.Bounds().Dy() {
				img.Set(x+dx, y, c)
			}
			if y+h-1 >= 0 && y+h-1 < img.Bounds().Dy() {
				img.Set(x+dx, y+h-1, c)
			}
		}
	}
	// Left and right
	for dy := 0; dy < h; dy++ {
		if y+dy < img.Bounds().Dy() && y+dy >= 0 {
			if x >= 0 && x < img.Bounds().Dx() {
				img.Set(x, y+dy, c)
			}
			if x+w-1 >= 0 && x+w-1 < img.Bounds().Dx() {
				img.Set(x+w-1, y+dy, c)
			}
		}
	}
}

func (ui *GameUI) drawFilledCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r*r {
				if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
					img.Set(x, y, c)
				}
			}
		}
	}
}

func (ui *GameUI) drawCircleBorder(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for angle := 0; angle < 360; angle++ {
		rad := float64(angle) * math.Pi / 180.0
		x := int(float64(cx) + float64(r)*math.Cos(rad))
		y := int(float64(cy) + float64(r)*math.Sin(rad))
		if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
			img.Set(x, y, c)
		}
	}
}

func (ui *GameUI) drawFilledDiamond(img *image.RGBA, cx, cy, w, h int, c color.RGBA) {
	for y := cy - h; y <= cy+h; y++ {
		for x := cx - w; x <= cx+w; x++ {
			dx := float64(x - cx)
			dy := float64(y - cy)
			if math.Abs(dx)/float64(w)+math.Abs(dy)/float64(h) <= 1.0 {
				if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
					img.Set(x, y, c)
				}
			}
		}
	}
}

func (ui *GameUI) drawDiamondBorder(img *image.RGBA, cx, cy, w, h int, c color.RGBA) {
	// Draw diamond outline
	points := [][]int{
		{cx, cy - h}, // Top
		{cx + w, cy}, // Right
		{cx, cy + h}, // Bottom
		{cx - w, cy}, // Left
	}
	for i := 0; i < len(points); i++ {
		p1 := points[i]
		p2 := points[(i+1)%len(points)]
		ui.drawLine(img, p1[0], p1[1], p2[0], p2[1], c)
	}
}

func (ui *GameUI) drawFilledRoundedRect(img *image.RGBA, x, y, w, h, r int, c color.RGBA) {
	// Draw main rectangle
	ui.drawFilledRect(img, x+r, y, w-2*r, h, c)
	ui.drawFilledRect(img, x, y+r, w, h-2*r, c)

	// Draw rounded corners (simplified - just draw circles)
	ui.drawFilledCircle(img, x+r, y+r, r, c)
	ui.drawFilledCircle(img, x+w-r, y+r, r, c)
	ui.drawFilledCircle(img, x+r, y+h-r, r, c)
	ui.drawFilledCircle(img, x+w-r, y+h-r, r, c)
}

func (ui *GameUI) drawRoundedRectBorder(img *image.RGBA, x, y, w, h, r int, c color.RGBA) {
	// Draw straight edges
	ui.drawLine(img, x+r, y, x+w-r, y, c)         // Top
	ui.drawLine(img, x+r, y+h-1, x+w-r, y+h-1, c) // Bottom
	ui.drawLine(img, x, y+r, x, y+h-r, c)         // Left
	ui.drawLine(img, x+w-1, y+r, x+w-1, y+h-r, c) // Right

	// Draw rounded corners (simplified)
	ui.drawCircleBorder(img, x+r, y+r, r, c)
	ui.drawCircleBorder(img, x+w-r, y+r, r, c)
	ui.drawCircleBorder(img, x+r, y+h-r, r, c)
	ui.drawCircleBorder(img, x+w-r, y+h-r, r, c)
}

func (ui *GameUI) drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := 1
	if x1 > x2 {
		sx = -1
	}
	sy := 1
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy

	x, y := x1, y1
	for {
		if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
			img.Set(x, y, c)
		}
		if x == x2 && y == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

func (ui *GameUI) drawTextCentered(img *image.RGBA, x, y int, text string, c color.RGBA) {
	// Simplified text rendering - just draw a line for each character
	// In a full implementation, you'd use proper text rendering
	textLen := len(text)
	startX := x - (textLen * 3)
	for i := range text {
		charX := startX + i*6
		// Draw a simple representation (just a few pixels)
		if charX >= 0 && charX < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
			img.Set(charX, y, c)
			if charX+1 < img.Bounds().Dx() {
				img.Set(charX+1, y, c)
			}
			if y+1 < img.Bounds().Dy() {
				img.Set(charX, y+1, c)
			}
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// gameLoop runs the game update loop
func (ui *GameUI) gameLoop() {
	ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
	defer ticker.Stop()

	for range ticker.C {
		// Handle continuous key presses and mouse movement
		if ui.game.State == StatePlaying {
			// Keyboard movement
			if ui.keysPressed[fyne.KeyLeft] {
				ui.game.MovePlayer(-1)
			}
			if ui.keysPressed[fyne.KeyRight] {
				ui.game.MovePlayer(1)
			}

			// Mouse movement (instant, keeps pace with cursor)
			if ui.mouseX >= 0 && ui.canvasWidth > 0 {
				// Convert mouse X to game coordinates
				// Ensure we use the full canvas width for accurate scaling
				scaleX := float64(GameWidth) / float64(ui.canvasWidth)
				gameX := float64(ui.mouseX) * scaleX

				// Center player on mouse position
				targetX := gameX - PlayerWidth/2

				// Clamp to game bounds (allow reaching right edge)
				if targetX < 0 {
					targetX = 0
				}
				if targetX+PlayerWidth > GameWidth {
					targetX = GameWidth - PlayerWidth
				}

				// Instant movement to mouse position (no smoothing)
				ui.game.Player.X = targetX
			}
		}

		ui.game.Update()

		// Play sound effects based on game state changes
		if ui.soundManager != nil {
			// Check for level complete
			if ui.game.State == StateLevelComplete && ui.game.Level > ui.prevLevel {
				ui.soundManager.PlayWheeHoo()
				ui.prevLevel = ui.game.Level
			}

			// Check for life lost
			if ui.game.Lives < ui.prevLives {
				ui.soundManager.PlayExplosion()
				ui.prevLives = ui.game.Lives
			}

			// Check for invader hit (score increased significantly)
			scoreDiff := ui.game.Score - ui.prevScore
			if scoreDiff >= 50 && ui.game.State == StatePlaying {
				// Score increased by at least 50 (invader hit)
				// Alternate between zip and boing
				if ui.invaderHitCount%2 == 0 {
					ui.soundManager.PlayZip()
				} else {
					ui.soundManager.PlayBoing()
				}
				ui.invaderHitCount++
				ui.prevScore = ui.game.Score
			} else if scoreDiff < 0 {
				// Score decreased (missed shot penalty or cheat key)
				ui.prevScore = ui.game.Score
			}
		}

		fyne.Do(ui.updateLabels)
		fyne.Do(ui.gameCanvas.Refresh)

		// Check for game over and save high score
		if ui.game.State == StateGameOver && ui.game.Score > 0 {
			if ui.highScoreMgr.AddScore(ui.game.Score) {
				fyne.Do(func() {
					ui.highScoreLabel.SetText("High Scores:\n" + ui.formatHighScores())
				})
			}
		}
	}
}

// updateLabels updates the UI labels
// Note: This function should be called via fyne.Do() when called from non-UI threads
func (ui *GameUI) updateLabels() {
	ui.scoreLabel.SetText("Score: " + strconv.Itoa(ui.game.Score))
	ui.livesLabel.SetText("Lives: " + strconv.Itoa(ui.game.Lives))
	ui.levelLabel.SetText("Level: " + strconv.Itoa(ui.game.Level))

	switch ui.game.State {
	case StateMenu:
		ui.statusLabel.SetText("Press Start to begin")
		ui.startButton.Enable()
		ui.pauseButton.Disable()
	case StatePlaying:
		ui.statusLabel.SetText("Playing...")
		ui.startButton.Disable()
		ui.pauseButton.Enable()
		ui.pauseButton.SetText("Pause")
	case StatePaused:
		ui.statusLabel.SetText("PAUSED - Press P to resume")
		ui.pauseButton.SetText("Resume")
	case StateGameOver:
		ui.statusLabel.SetText("GAME OVER - Press Start to play again")
		ui.startButton.Enable()
		ui.pauseButton.Disable()
	case StateLevelComplete:
		ui.statusLabel.SetText("Level Complete! Next level starting...")
	}
}

// formatHighScores formats high scores for display
func (ui *GameUI) formatHighScores() string {
	scores := ui.highScoreMgr.GetHighScores()
	if len(scores) == 0 {
		return "No high scores yet"
	}

	var sb strings.Builder
	for i, score := range scores {
		sb.WriteString(strconv.Itoa(i+1) + ". " + strconv.Itoa(score) + "\n")
	}
	return strings.TrimSpace(sb.String())
}

// Event handlers

func (ui *GameUI) onStart() {
	ui.game.Start()
	// Reset sound tracking
	ui.prevLives = ui.game.Lives
	ui.prevLevel = ui.game.Level
	ui.prevScore = ui.game.Score
	ui.invaderHitCount = 0
	ui.updateLabels()
}

func (ui *GameUI) onPause() {
	ui.game.Pause()
	ui.updateLabels()
}

func (ui *GameUI) onResetScores() {
	// If dialog already exists, bring it to front
	if ui.resetDialog != nil {
		ui.resetDialog.RequestFocus()
		ui.resetDialog.Show()
		return
	}

	// Create confirmation dialog with text input
	confirmText := widget.NewEntry()
	confirmText.SetPlaceHolder("Type 'KrankyBear' to confirm")

	confirmLabel := widget.NewLabel("Are you sure you want to reset all high scores?\n\nType 'KrankyBear' to confirm:")

	var confirmButton *widget.Button
	confirmButton = widget.NewButton("Reset", func() {
		if confirmText.Text == "KrankyBear" {
			ui.highScoreMgr.ResetHighScores()
			ui.highScoreLabel.SetText("High Scores:\n" + ui.formatHighScores())
			// Close the dialog
			if ui.resetDialog != nil {
				ui.resetDialog.Close()
				ui.resetDialog = nil
			}
		}
	})
	confirmButton.Disable()

	cancelButton := widget.NewButton("Cancel", func() {
		if ui.resetDialog != nil {
			ui.resetDialog.Close()
			ui.resetDialog = nil
		}
	})

	// Enable confirm button only when text matches
	confirmText.OnChanged = func(text string) {
		if text == "KrankyBear" {
			confirmButton.Enable()
		} else {
			confirmButton.Disable()
		}
	}

	content := container.NewVBox(
		confirmLabel,
		confirmText,
		container.NewHBox(confirmButton, cancelButton),
	)

	// Create a custom dialog window
	dialogWindow := ui.app.NewWindow("Confirm Reset High Scores")
	dialogWindow.SetIcon(resourceKrankyBearUSArmyCombatHelmetWoodlandPng)
	dialogWindow.SetContent(content)
	dialogWindow.Resize(fyne.NewSize(400, 200))

	// Track the dialog window
	ui.resetDialog = dialogWindow

	// Reset tracking when window closes
	dialogWindow.SetOnClosed(func() {
		ui.resetDialog = nil
	})

	// Position dialog relative to main window (after showing)
	ui.positionDialogRelativeToMain(dialogWindow)
}

func (ui *GameUI) showAboutDialog() {
	// If dialog already exists, bring it to front
	if ui.aboutDialog != nil {
		ui.aboutDialog.RequestFocus()
		ui.aboutDialog.Show()
		return
	}

	// Create a new window for the dialog
	dialogWindow := ui.app.NewWindow("About " + appName)
	dialogWindow.SetIcon(resourceKrankyBearUSArmyCombatHelmetWoodlandPng)
	dialogWindow.Resize(fyne.NewSize(500, 300))
	dialogWindow.SetFixedSize(true)

	// Track the dialog window
	ui.aboutDialog = dialogWindow

	// Reset tracking when window closes
	dialogWindow.SetOnClosed(func() {
		ui.aboutDialog = nil
	})

	// Create image based on version status
	// USArmyCombatHelmetWoodland: current version or update available
	// Could use different image for newer version if needed
	var iconResource fyne.Resource
	if ui.versionStatus == "newer" {
		iconResource = resourceKrankyBearUSArmyCombatHelmetWoodlandPng
	} else {
		// "current", "update", or "unknown" - show USArmyCombatHelmetWoodland
		iconResource = resourceKrankyBearUSArmyCombatHelmetWoodlandPng
	}
	iconImage := canvas.NewImageFromResource(iconResource)
	iconImage.FillMode = canvas.ImageFillContain
	iconImage.SetMinSize(fyne.NewSize(150, 150))

	// Create GitHub URL
	githubLink, err := url.Parse("https://github.com/amarillier/KrankyBearThreatInvaders")
	if err != nil {
		fyne.LogError("Could not parse URL", err)
	}
	githubHyperlink := widget.NewHyperlink("https://github.com/amarillier/KrankyBearThreatInvaders", githubLink)
	githubHyperlink.Alignment = fyne.TextAlignLeading

	// Create text content
	aboutText := widget.NewRichTextFromMarkdown(`# ` + appName + `

**Version:** ` + appVersion + `

**Author:** ` + appAuthor + `

**Copyright:** ` + appCopyright + `

A Space Invaders-like game where you defend against CVE and KB threats!

Enjoy the game!`)

	aboutText.Wrapping = fyne.TextWrapWord

	// Layout: image on left, text on right (left-justified)
	textContainer := container.NewVBox(
		container.NewPadded(aboutText),
		githubHyperlink,
	)
	content := container.NewBorder(
		nil,
		container.NewCenter(widget.NewButton("Close", func() {
			dialogWindow.Close()
		})),
		nil,
		nil,
		container.NewHBox(
			container.NewPadded(iconImage),
			textContainer,
		),
	)

	dialogWindow.SetContent(content)
	// Position dialog relative to main window (after showing)
	ui.positionDialogRelativeToMain(dialogWindow)
}

func (ui *GameUI) showHelpDialog() {
	// If dialog already exists, bring it to front
	if ui.helpDialog != nil {
		ui.helpDialog.RequestFocus()
		ui.helpDialog.Show()
		return
	}

	// Create a new window for the dialog
	dialogWindow := ui.app.NewWindow("Help - " + appName)
	dialogWindow.SetIcon(resourceKrankyBearUSArmyCombatHelmetWoodlandPng)
	dialogWindow.Resize(fyne.NewSize(600, 500))
	dialogWindow.SetFixedSize(true)

	// Track the dialog window
	ui.helpDialog = dialogWindow

	// Reset tracking when window closes
	dialogWindow.SetOnClosed(func() {
		ui.helpDialog = nil
	})

	// Create image based on version status
	// USArmyCombatHelmetWoodland: current version or update available
	// Could use different image for newer version if needed
	var iconResource fyne.Resource
	if ui.versionStatus == "newer" {
		iconResource = resourceKrankyBearUSArmyCombatHelmetWoodlandPng
	} else {
		// "current", "update", or "unknown" - show USArmyCombatHelmetWoodland
		iconResource = resourceKrankyBearUSArmyCombatHelmetWoodlandPng
	}
	iconImage := canvas.NewImageFromResource(iconResource)
	iconImage.FillMode = canvas.ImageFillContain
	iconImage.SetMinSize(fyne.NewSize(150, 150))

	// Create text content using RichTextFromMarkdown like Tetris
	helpText := widget.NewRichTextFromMarkdown(`# How to Play

## Controls

**Arrow Keys:**
- Left/Right arrows: Move defender left/right

**Space Bar:** Fire bullets at invaders

**Mouse:**
- Move mouse left/right to control defender
- Click to fire bullets

**P:** Pause/Resume the game

**F12:** Boss key - pause the game and hide the window (quickly hide the game)

**Z:** Cheat key - add one life (deducts 100 points per life)

## Gameplay

- Defend against CVE and KB threats (invaders)
- CVEs (red squares/diamonds) are worth 100 points
- KBs (blue circles/rounded) are worth 50 points
- Neutralize enemy bullets with your bullets for 5 points
- Lives depend on difficulty: Normal (3), Medium (5), Easy (8)
- Game gets harder as you progress through levels

## Difficulty Modes

- **Normal:** 3 lives (default)
- **Medium:** 5 lives
- **Easy:** 8 lives

Select difficulty from the menu before starting a new game.

## Scoring

- Hit invaders to score points
- Miss shots lose 10 points
- Level bonus: +10 points per level`)

	helpText.Wrapping = fyne.TextWrapWord

	// Layout: image on left, text on right (left-justified) - match About pattern exactly
	// Use scroll container for the text to ensure proper rendering
	// Make scroll container expand to fill available height
	textScroll := container.NewScroll(helpText)
	textScroll.SetMinSize(fyne.NewSize(400, 0))

	// Use Border with scroll in center to make it expand to full height
	textContainer := container.NewBorder(
		nil,
		nil,
		nil,
		nil,
		container.NewPadded(textScroll),
	)

	content := container.NewBorder(
		nil,
		container.NewCenter(widget.NewButton("Close", func() {
			dialogWindow.Close()
		})),
		nil,
		nil,
		container.NewHBox(
			container.NewPadded(iconImage),
			textContainer,
		),
	)

	dialogWindow.SetContent(content)
	// Ensure window is fully initialized before showing
	// Show first, then center (ensures it appears on same display as main window)
	dialogWindow.Show()
	dialogWindow.CenterOnScreen()
}

// positionDialogRelativeToMain positions a dialog window relative to the main window
// Uses the same pattern as KrankyBearTetris: show first, then center
// This ensures the dialog appears on the same display as the main window
// NOTE: Window should already have content set before calling this
func (ui *GameUI) positionDialogRelativeToMain(dialogWindow fyne.Window) {
	// Show the dialog first (it will appear on the same display as the main window)
	// This must be done after SetContent() to ensure the window has valid size
	dialogWindow.Show()

	// Center on screen - this will center on the same display as the main window
	// because we showed it first, it inherits the display context
	dialogWindow.CenterOnScreen()
}

func (ui *GameUI) showUpdateDialog() {
	// Check if window already exists
	if ui.updateDialog != nil {
		ui.updateDialog.RequestFocus()
		ui.updateDialog.Show()
		return
	}

	// Run update check in goroutine to avoid blocking UI
	go func() {
		uc := updatechecker.New("amarillier", "KrankyBearThreatInvaders", "KrankyBear ThreatInvaders", "https://github.com/amarillier/KrankyBearThreatInvaders/releases/latest", 0, false)
		uc.CheckForUpdate(appVersion)
		fyne.Do(func() {
			ui.updateAlert(uc.Message)
		})
	}()
}

// updateAlert displays the update check result in a dialog window
func (ui *GameUI) updateAlert(updtmsg string) {
	// Check if window already exists
	if ui.updateDialog != nil {
		ui.updateDialog.RequestFocus()
		ui.updateDialog.Show()
		return
	}

	// Create release link
	releaselink, rerr := url.Parse("https://github.com/amarillier/KrankyBearThreatInvaders/releases/latest")
	if rerr != nil {
		fyne.LogError("Could not parse URL", rerr)
	}
	myreleaselink := widget.NewHyperlink("https://github.com/amarillier/KrankyBearThreatInvaders/releases/latest", releaselink)
	myreleaselink.Alignment = fyne.TextAlignLeading

	// Create release notes link
	releasenoteslink, rnerr := url.Parse("https://github.com/amarillier/KrankyBearThreatInvaders/blob/allanm/ReleaseNotes.txt")
	if rnerr != nil {
		fyne.LogError("Could not parse URL", rnerr)
	}
	myreleasenoteslink := widget.NewHyperlink("https://github.com/amarillier/KrankyBearThreatInvaders/blob/allanm/ReleaseNotes.txt", releasenoteslink)
	myreleasenoteslink.Alignment = fyne.TextAlignLeading

	// Create image based on update message
	// Hard Hat: running newer version than published
	// USArmyCombatHelmetWoodland: current version or update available
	var kbimg *canvas.Image
	if strings.Contains(updtmsg, "newer version") {
		// Running a newer version than published - show Hard Hat
		kbimg = canvas.NewImageFromResource(resourceKrankyBearHardHatPng)
		kbimg.FillMode = canvas.ImageFillContain
	} else if strings.Contains(updtmsg, "running the latest") {
		// Running the latest published version - show Helmet
		kbimg = canvas.NewImageFromResource(resourceKrankyBearUSArmyCombatHelmetWoodlandPng)
		kbimg.FillMode = canvas.ImageFillContain
	} else {
		// Update available or unknown status - show Helmet
		kbimg = canvas.NewImageFromResource(resourceKrankyBearUSArmyCombatHelmetWoodlandPng)
		kbimg.FillMode = canvas.ImageFillContain
	}
	kbimg.SetMinSize(fyne.NewSize(150, 150))

	// Create text label with update message - clean up excessive line breaks
	cleanedMsg := strings.ReplaceAll(updtmsg, "======", "")
	cleanedMsg = strings.ReplaceAll(cleanedMsg, "=====", "")
	cleanedMsg = strings.ReplaceAll(cleanedMsg, "====", "")
	cleanedMsg = strings.TrimSpace(cleanedMsg)
	text := widget.NewLabel(cleanedMsg)
	text.Wrapping = fyne.TextWrapWord

	// Layout: image on left, text on right (left-justified)
	textContainer := container.NewVBox(
		container.NewPadded(text),
		myreleaselink,
		myreleasenoteslink,
	)
	content := container.NewBorder(
		nil,
		container.NewCenter(widget.NewButton("Close", func() {
			if ui.updateDialog != nil {
				ui.updateDialog.Close()
			}
		})),
		nil,
		nil,
		container.NewHBox(
			container.NewPadded(kbimg),
			textContainer,
		),
	)

	// Create window
	// ui.updateDialog = ui.app.NewWindow(appName + ": Update Check")
	ui.updateDialog = ui.app.NewWindow("KrankyBear ThreatInvaders: Update Check")
	ui.updateDialog.SetIcon(resourceKrankyBearUSArmyCombatHelmetWoodlandPng)
	ui.updateDialog.Resize(fyne.NewSize(700, 300))
	ui.updateDialog.SetContent(content)
	ui.updateDialog.SetOnClosed(func() {
		ui.updateDialog = nil
	})
	// Position dialog relative to main window (after showing)
	ui.positionDialogRelativeToMain(ui.updateDialog)
}

// setupSystemTray sets up the system tray icon and menu using desktop.App interface
func (ui *GameUI) setupSystemTray() {
	// Check if app supports desktop features (system tray)
	if desk, ok := ui.app.(desktop.App); ok {
		// Create menu items
		ui.showMenuItem = fyne.NewMenuItem("Show", func() {
			fyne.Do(ui.showWindow)
		})
		ui.hideMenuItem = fyne.NewMenuItem("Hide", func() {
			fyne.Do(ui.hideWindow)
		})
		fyne.NewMenuItemSeparator()
		helpTray := fyne.NewMenuItem("Help", func() {
			fyne.Do(ui.showHelpDialog)
		})
		aboutTray := fyne.NewMenuItem("About", func() {
			fyne.Do(ui.showAboutDialog)
		})
		updateTray := fyne.NewMenuItem("Check for Update", func() {
			ui.showUpdateDialog()
		})

		// Create difficulty menu items for system tray
		normalTray := fyne.NewMenuItem("Normal (3 lives)", func() {
			fyne.Do(func() {
				ui.setDifficulty(DifficultyNormal)
				ui.updateTrayMenuState()
			})
		})
		mediumTray := fyne.NewMenuItem("Medium (5 lives)", func() {
			fyne.Do(func() {
				ui.setDifficulty(DifficultyMedium)
				ui.updateTrayMenuState()
			})
		})
		easyTray := fyne.NewMenuItem("Easy (8 lives)", func() {
			fyne.Do(func() {
				ui.setDifficulty(DifficultyEasy)
				ui.updateTrayMenuState()
			})
		})

		// Set initial checked state
		normalTray.Checked = (ui.difficultyMode == DifficultyNormal)
		mediumTray.Checked = (ui.difficultyMode == DifficultyMedium)
		easyTray.Checked = (ui.difficultyMode == DifficultyEasy)

		difficultySubMenuTray := fyne.NewMenu("Difficulty",
			normalTray,
			mediumTray,
			easyTray,
		)
		difficultyMenuItemTray := fyne.NewMenuItem("Difficulty", nil)
		difficultyMenuItemTray.ChildMenu = difficultySubMenuTray

		// Sound toggle menu item for system tray
		soundTray := fyne.NewMenuItem("Sound Effects", func() {
			fyne.Do(func() {
				ui.toggleSound()
				ui.updateTrayMenuState()
			})
		})
		soundTray.Checked = (ui.soundManager != nil && ui.soundManager.IsEnabled())

		fyne.NewMenuItemSeparator()
		quitTray := fyne.NewMenuItem("Quit", func() {
			fyne.Do(ui.app.Quit)
		})

		// Create menu
		menu := fyne.NewMenu("KrankyBear ThreatInvaders",
			ui.showMenuItem,
			ui.hideMenuItem,
			fyne.NewMenuItemSeparator(),
			difficultyMenuItemTray,
			soundTray,
			fyne.NewMenuItemSeparator(),
			aboutTray,
			helpTray,
			updateTray,
			fyne.NewMenuItemSeparator(),
			quitTray,
		)

		// Set system tray menu and icon
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(resourceKrankyBearUSArmyCombatHelmetWoodlandPng)

		// Update menu state based on window visibility
		ui.updateTrayMenuState()
	}
}

// updateTrayMenuState updates the tray menu items based on window visibility
func (ui *GameUI) updateTrayMenuState() {
	// Recreate the menu with updated state
	if desk, ok := ui.app.(desktop.App); ok {
		// Create menu items
		aboutTray := fyne.NewMenuItem("About", func() {
			fyne.Do(ui.showAboutDialog)
		})
		helpTray := fyne.NewMenuItem("Help", func() {
			fyne.Do(ui.showHelpDialog)
		})
		updateTray := fyne.NewMenuItem("Check for Update", func() {
			ui.showUpdateDialog()
		})

		// Create difficulty menu items for system tray
		normalTray := fyne.NewMenuItem("Normal (3 lives)", func() {
			fyne.Do(func() {
				ui.setDifficulty(DifficultyNormal)
				ui.updateTrayMenuState()
			})
		})
		mediumTray := fyne.NewMenuItem("Medium (5 lives)", func() {
			fyne.Do(func() {
				ui.setDifficulty(DifficultyMedium)
				ui.updateTrayMenuState()
			})
		})
		easyTray := fyne.NewMenuItem("Easy (8 lives)", func() {
			fyne.Do(func() {
				ui.setDifficulty(DifficultyEasy)
				ui.updateTrayMenuState()
			})
		})

		// Set checked state based on current difficulty
		normalTray.Checked = (ui.difficultyMode == DifficultyNormal)
		mediumTray.Checked = (ui.difficultyMode == DifficultyMedium)
		easyTray.Checked = (ui.difficultyMode == DifficultyEasy)

		difficultySubMenuTray := fyne.NewMenu("Difficulty",
			normalTray,
			mediumTray,
			easyTray,
		)
		difficultyMenuItemTray := fyne.NewMenuItem("Difficulty", nil)
		difficultyMenuItemTray.ChildMenu = difficultySubMenuTray

		// Sound toggle menu item for system tray
		soundTray := fyne.NewMenuItem("Sound Effects", func() {
			fyne.Do(func() {
				ui.toggleSound()
				ui.updateTrayMenuState()
			})
		})
		soundTray.Checked = (ui.soundManager != nil && ui.soundManager.IsEnabled())

		// Create show/hide menu items - always create both, but we'll handle logic in callbacks
		ui.showMenuItem = fyne.NewMenuItem("Show", func() {
			fyne.Do(func() {
				if !ui.windowVisible {
					ui.showWindow()
				}
			})
		})
		ui.hideMenuItem = fyne.NewMenuItem("Hide", func() {
			fyne.Do(func() {
				if ui.windowVisible {
					ui.hideWindow()
				}
			})
		})

		// Set enabled/disabled state
		ui.showMenuItem.Disabled = ui.windowVisible
		ui.hideMenuItem.Disabled = !ui.windowVisible

		// Recreate the menu with updated state
		menu := fyne.NewMenu("KrankyBear ThreatInvaders",
			ui.showMenuItem,
			ui.hideMenuItem,
			fyne.NewMenuItemSeparator(),
			difficultyMenuItemTray,
			soundTray,
			fyne.NewMenuItemSeparator(),
			aboutTray,
			helpTray,
			updateTray,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {
				fyne.Do(ui.app.Quit)
			}),
		)
		desk.SetSystemTrayMenu(menu)
	}
}

func (ui *GameUI) showWindow() {
	ui.window.Show()
	ui.windowVisible = true
	ui.updateTrayMenuState()
}

func (ui *GameUI) hideWindow() {
	ui.window.Hide()
	ui.windowVisible = false
	ui.updateTrayMenuState()
}

// Show shows the window
func (ui *GameUI) Show() {
	ui.window.Show()
	ui.windowVisible = true
	ui.updateTrayMenuState()
}

// "Now this is not the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
