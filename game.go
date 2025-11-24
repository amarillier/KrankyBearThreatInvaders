package main

import (
	"math/rand"
	"time"
)

// Game dimensions
const (
	GameWidth  = 800
	GameHeight = 600
	PlayerY    = GameHeight - 60
	InvaderRows = 5
	InvaderCols = 10
	InvaderWidth = 50
	InvaderHeight = 40
	PlayerWidth = 60
	PlayerHeight = 40
	BulletWidth = 4
	BulletHeight = 10
	InvaderBulletWidth = 4
	InvaderBulletHeight = 10
)

// GameState represents the current state of the game
type GameState int

const (
	StateMenu GameState = iota
	StatePlaying
	StatePaused
	StateGameOver
	StateLevelComplete
)

// DifficultyMode represents the game difficulty level
type DifficultyMode int

const (
	DifficultyNormal DifficultyMode = iota
	DifficultyMedium
	DifficultyEasy
)

// GetLives returns the starting number of lives for a difficulty mode
func (d DifficultyMode) GetLives() int {
	switch d {
	case DifficultyEasy:
		return 8
	case DifficultyMedium:
		return 5
	case DifficultyNormal:
		return 3
	default:
		return 3
	}
}

// String returns the string representation of the difficulty mode
func (d DifficultyMode) String() string {
	switch d {
	case DifficultyEasy:
		return "Easy"
	case DifficultyMedium:
		return "Medium"
	case DifficultyNormal:
		return "Normal"
	default:
		return "Normal"
	}
}

// InvaderType represents the visual type of invader
type InvaderType int

const (
	InvaderTypeCVE InvaderType = iota // CVE invaders (red, square/rectangular)
	InvaderTypeKB                      // KB invaders (blue, circular/rounded)
)

// Invader represents a CVE/KB invader
type Invader struct {
	X          float64
	Y          float64
	Width      float64
	Height     float64
	Text       string      // CVE ID or KB number
	Type       InvaderType // Visual type (CVE or KB)
	IsCVE      bool        // true for CVE, false for KB (for scoring)
	Destroyed  bool
	Speed      float64
	Direction  int // 1 for right, -1 for left
	Color      [3]uint8 // RGB color for rendering
	Shape      string   // "square", "circle", "diamond" for different appearances
}

// Bullet represents a bullet
type Bullet struct {
	X         float64
	Y         float64
	Width     float64
	Height    float64
	Speed     float64
	IsPlayer  bool // true for player bullet, false for invader bullet
	Active    bool
}

// Player represents the defender
type Player struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
	Speed  float64
}

// Game represents the Threat Invaders game
type Game struct {
	Player          *Player
	Invaders        [][]*Invader
	PlayerBullets   []*Bullet
	InvaderBullets  []*Bullet
	Score           int
	Lives           int
	Level           int
	State           GameState
	Difficulty      DifficultyMode
	InvaderSpeed    float64
	BulletSpeed     float64
	InvaderDirection int // 1 for right, -1 for left
	LastMoveTime    time.Time
	LastShootTime   time.Time
	LastInvaderShoot time.Time
	ShootCooldown   time.Duration
	InvaderShootCooldown time.Duration
	MissedShots     int // Track missed shots for point deduction
	InvadersReachedBottom int // Track invaders that reached bottom
	rng             *rand.Rand
	cveData         []string
	kbData          []string
}

// NewGame creates a new game instance
func NewGame(cveData, kbData []string, difficulty DifficultyMode) *Game {
	g := &Game{
		Player: &Player{
			X:      GameWidth / 2,
			Y:      PlayerY,
			Width:  PlayerWidth,
			Height: PlayerHeight,
			Speed:  10.0, // Increased from 5.0 for more responsive movement
		},
		Score:           0,
		Lives:           difficulty.GetLives(),
		Level:           1,
		State:           StateMenu,
		Difficulty:      difficulty,
		InvaderSpeed:    1.0,
		BulletSpeed:     5.0,
		InvaderDirection: 1,
		ShootCooldown:   200 * time.Millisecond,
		InvaderShootCooldown: 2 * time.Second,
		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
		cveData:         cveData,
		kbData:          kbData,
	}
	g.InitializeLevel()
	return g
}

// InitializeLevel sets up a new level
func (g *Game) InitializeLevel() {
	// Clear existing invaders and bullets
	g.Invaders = make([][]*Invader, InvaderRows)
	g.PlayerBullets = make([]*Bullet, 0)
	g.InvaderBullets = make([]*Bullet, 0)
	g.MissedShots = 0
	g.InvadersReachedBottom = 0
	
	// Create invaders in formation
	startX := 100.0
	startY := 50.0
	spacingX := 60.0
	spacingY := 50.0
	
	for row := 0; row < InvaderRows; row++ {
		g.Invaders[row] = make([]*Invader, InvaderCols)
		for col := 0; col < InvaderCols; col++ {
			// Alternate between CVE and KB
			isCVE := (row+col)%2 == 0
			var text string
			var invaderType InvaderType
			var color [3]uint8
			var shape string
			
			if isCVE && len(g.cveData) > 0 {
				text = g.cveData[(row*InvaderCols+col)%len(g.cveData)]
				invaderType = InvaderTypeCVE
				// CVEs: Red/orange colors, square/rectangular shape
				color = [3]uint8{220, 53, 69} // Red (#DC3545)
				shape = "square"
			} else if !isCVE && len(g.kbData) > 0 {
				text = g.kbData[(row*InvaderCols+col)%len(g.kbData)]
				invaderType = InvaderTypeKB
				// KBs: Blue colors, circular/rounded shape
				color = [3]uint8{0, 123, 255} // Blue (#007BFF)
				shape = "circle"
			} else {
				// Fallback if no data
				if isCVE {
					text = "CVE-XXXX-XXXX"
					invaderType = InvaderTypeCVE
					color = [3]uint8{220, 53, 69}
					shape = "square"
				} else {
					text = "KBXXXXXXX"
					invaderType = InvaderTypeKB
					color = [3]uint8{0, 123, 255}
					shape = "circle"
				}
			}
			
			// Add some variation to colors and shapes for visual interest
			if invaderType == InvaderTypeCVE {
				// Vary red shades slightly
				colorVariation := uint8((row*InvaderCols + col) % 30)
				if color[0] > colorVariation {
					color[0] -= colorVariation
				}
				// Some CVEs can be diamond-shaped
				if (row+col)%3 == 0 {
					shape = "diamond"
				}
			} else {
				// Vary blue shades slightly
				colorVariation := uint8((row*InvaderCols + col) % 30)
				if color[2] > colorVariation {
					color[2] -= colorVariation
				}
				// Some KBs can be rounded square
				if (row+col)%3 == 0 {
					shape = "rounded"
				}
			}
			
			g.Invaders[row][col] = &Invader{
				X:         startX + float64(col)*spacingX,
				Y:         startY + float64(row)*spacingY,
				Width:     InvaderWidth,
				Height:    InvaderHeight,
				Text:      text,
				Type:      invaderType,
				IsCVE:     isCVE,
				Destroyed: false,
				Speed:     g.InvaderSpeed,
				Direction: 1,
				Color:     color,
				Shape:     shape,
			}
		}
	}
	
	// Increase difficulty with level
	g.InvaderSpeed = 1.0 + float64(g.Level-1)*0.2
	g.BulletSpeed = 5.0 + float64(g.Level-1)*0.5
	g.InvaderShootCooldown = time.Duration(2000 - (g.Level-1)*100) * time.Millisecond
	if g.InvaderShootCooldown < 500*time.Millisecond {
		g.InvaderShootCooldown = 500 * time.Millisecond
	}
	
	g.LastMoveTime = time.Now()
	g.LastShootTime = time.Now()
	g.LastInvaderShoot = time.Now()
}

// MovePlayer moves the player left or right
func (g *Game) MovePlayer(direction int) {
	if g.State != StatePlaying {
		return
	}
	
	newX := g.Player.X + float64(direction)*g.Player.Speed
	if newX >= 0 && newX+PlayerWidth <= GameWidth {
		g.Player.X = newX
	}
}

// Shoot creates a new player bullet
func (g *Game) Shoot() {
	if g.State != StatePlaying {
		return
	}
	
	now := time.Now()
	if now.Sub(g.LastShootTime) < g.ShootCooldown {
		return
	}
	
	bullet := &Bullet{
		X:        g.Player.X + PlayerWidth/2 - BulletWidth/2,
		Y:        g.Player.Y,
		Width:    BulletWidth,
		Height:   BulletHeight,
		Speed:    g.BulletSpeed,
		IsPlayer: true,
		Active:   true,
	}
	
	g.PlayerBullets = append(g.PlayerBullets, bullet)
	g.LastShootTime = now
}

// Update updates the game state
func (g *Game) Update() {
	if g.State != StatePlaying {
		return
	}
	
	now := time.Now()
	
	// Move invaders
	if now.Sub(g.LastMoveTime) > 50*time.Millisecond {
		g.moveInvaders()
		g.LastMoveTime = now
	}
	
	// Update bullets
	g.updateBullets()
	
	// Check collisions
	g.checkCollisions()
	
	// Invader shooting
	if now.Sub(g.LastInvaderShoot) > g.InvaderShootCooldown {
		g.invaderShoot()
		g.LastInvaderShoot = now
	}
	
	// Check for level complete
	if g.allInvadersDestroyed() {
		g.State = StateLevelComplete
		g.Level++
		time.AfterFunc(2*time.Second, func() {
			if g.State == StateLevelComplete {
				g.InitializeLevel()
				g.State = StatePlaying
			}
		})
	}
	
	// Check for game over
	if g.Lives <= 0 || g.invadersReachedBottom() {
		g.State = StateGameOver
	}
}

// moveInvaders moves all active invaders
func (g *Game) moveInvaders() {
	shouldMoveDown := false
	leftmostX := float64(GameWidth)
	rightmostX := 0.0
	
	// Find boundaries of active invaders
	for _, row := range g.Invaders {
		for _, invader := range row {
			if invader != nil && !invader.Destroyed {
				if invader.X < leftmostX {
					leftmostX = invader.X
				}
				if invader.X+invader.Width > rightmostX {
					rightmostX = invader.X + invader.Width
				}
			}
		}
	}
	
	// Check if we need to move down and reverse direction
	if (g.InvaderDirection == 1 && rightmostX >= GameWidth-20) ||
		(g.InvaderDirection == -1 && leftmostX <= 20) {
		shouldMoveDown = true
		g.InvaderDirection *= -1
	}
	
	// Move invaders
	for _, row := range g.Invaders {
		for _, invader := range row {
			if invader != nil && !invader.Destroyed {
				if shouldMoveDown {
					invader.Y += 20
				} else {
					invader.X += float64(g.InvaderDirection) * invader.Speed
				}
			}
		}
	}
}

// updateBullets updates all bullet positions
func (g *Game) updateBullets() {
	// Update player bullets
	for _, bullet := range g.PlayerBullets {
		if bullet != nil && bullet.Active {
			bullet.Y -= bullet.Speed
			if bullet.Y < 0 {
				bullet.Active = false
				g.MissedShots++
				// Deduct points for missing
				g.Score -= 10
				if g.Score < 0 {
					g.Score = 0
				}
			}
		}
	}
	
	// Update invader bullets
	for _, bullet := range g.InvaderBullets {
		if bullet != nil && bullet.Active {
			bullet.Y += bullet.Speed
			if bullet.Y > GameHeight {
				bullet.Active = false
			}
		}
	}
	
	// Clean up inactive bullets
	g.PlayerBullets = g.cleanBullets(g.PlayerBullets)
	g.InvaderBullets = g.cleanBullets(g.InvaderBullets)
}

// checkCollisions checks for collisions between bullets and invaders/player
func (g *Game) checkCollisions() {
	// Check player bullets vs invader bullets (neutralize them)
	for _, pBullet := range g.PlayerBullets {
		if pBullet == nil || !pBullet.Active {
			continue
		}
		
		// Check collision with invader bullets first
		bulletHit := false
		for _, iBullet := range g.InvaderBullets {
			if iBullet == nil || !iBullet.Active {
				continue
			}
			
			// Check collision between player bullet and invader bullet
			if pBullet.X < iBullet.X+iBullet.Width &&
				pBullet.X+pBullet.Width > iBullet.X &&
				pBullet.Y < iBullet.Y+iBullet.Height &&
				pBullet.Y+pBullet.Height > iBullet.Y {
				
				// Bullets neutralize each other
				pBullet.Active = false
				iBullet.Active = false
				// Award small points for neutralizing enemy bullets
				g.Score += 5
				bulletHit = true
				break // Move to next player bullet
			}
		}
		
		if bulletHit {
			continue
		}
		
		// Check player bullets vs invaders
		for _, row := range g.Invaders {
			for _, invader := range row {
				if invader != nil && !invader.Destroyed {
					if pBullet.X < invader.X+invader.Width &&
						pBullet.X+pBullet.Width > invader.X &&
						pBullet.Y < invader.Y+invader.Height &&
						pBullet.Y+pBullet.Height > invader.Y {
						
						// Hit!
						invader.Destroyed = true
						pBullet.Active = false
						
						// Award points (CVEs worth more)
						if invader.IsCVE {
							g.Score += 100
						} else {
							g.Score += 50
						}
						
						// Bonus for level
						g.Score += g.Level * 10
						break // Move to next player bullet
					}
				}
			}
		}
	}
	
	// Clean up nil bullets
	g.PlayerBullets = g.cleanBullets(g.PlayerBullets)
	g.InvaderBullets = g.cleanBullets(g.InvaderBullets)
	
	// Check invader bullets vs player
	for _, bullet := range g.InvaderBullets {
		if bullet == nil || !bullet.Active {
			continue
		}
		
		if bullet.X < g.Player.X+g.Player.Width &&
			bullet.X+bullet.Width > g.Player.X &&
			bullet.Y < g.Player.Y+g.Player.Height &&
			bullet.Y+bullet.Height > g.Player.Y {
			
			// Hit player!
			bullet.Active = false
			g.Lives--
			if g.Lives <= 0 {
				g.State = StateGameOver
			}
		}
	}
}

// cleanBullets removes inactive and nil bullets from a slice
func (g *Game) cleanBullets(bullets []*Bullet) []*Bullet {
	activeBullets := make([]*Bullet, 0, len(bullets))
	for _, bullet := range bullets {
		if bullet != nil && bullet.Active {
			activeBullets = append(activeBullets, bullet)
		}
	}
	return activeBullets
}

// invaderShoot makes a random invader shoot
func (g *Game) invaderShoot() {
	// Collect all active invaders
	activeInvaders := make([]*Invader, 0)
	for _, row := range g.Invaders {
		for _, invader := range row {
			if invader != nil && !invader.Destroyed {
				activeInvaders = append(activeInvaders, invader)
			}
		}
	}
	
	if len(activeInvaders) == 0 {
		return
	}
	
	// Random invader shoots
	shooter := activeInvaders[g.rng.Intn(len(activeInvaders))]
	
	bullet := &Bullet{
		X:        shooter.X + shooter.Width/2 - InvaderBulletWidth/2,
		Y:        shooter.Y + shooter.Height,
		Width:    InvaderBulletWidth,
		Height:   InvaderBulletHeight,
		Speed:    2.0 + float64(g.Level-1)*0.3,
		IsPlayer: false,
		Active:   true,
	}
	
	g.InvaderBullets = append(g.InvaderBullets, bullet)
}

// allInvadersDestroyed checks if all invaders are destroyed
func (g *Game) allInvadersDestroyed() bool {
	for _, row := range g.Invaders {
		for _, invader := range row {
			if invader != nil && !invader.Destroyed {
				return false
			}
		}
	}
	return true
}

// invadersReachedBottom checks if any invader reached the bottom
func (g *Game) invadersReachedBottom() bool {
	for _, row := range g.Invaders {
		for _, invader := range row {
			if invader != nil && !invader.Destroyed && invader.Y+invader.Height >= PlayerY-10 {
				return true
			}
		}
	}
	return false
}

// Start starts a new game
func (g *Game) Start() {
	g.Score = 0
	g.Lives = g.Difficulty.GetLives()
	g.Level = 1
	g.State = StatePlaying
	g.InitializeLevel()
}

// AddLife adds one life (cheat key) and deducts 100 points
func (g *Game) AddLife() {
	g.Lives++
	g.Score -= 100
	if g.Score < 0 {
		g.Score = 0
	}
}

// Pause toggles pause state
func (g *Game) Pause() {
	if g.State == StatePlaying {
		g.State = StatePaused
	} else if g.State == StatePaused {
		g.State = StatePlaying
	}
}

// "Now this is not the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942

