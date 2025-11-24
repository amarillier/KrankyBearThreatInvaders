package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2/app"
)

const (
	// appName    = "KrankyBear ThreatInvaders"
	appVersion = "0.1.0" // see FyneApp.toml
	appAuthor  = "Allan Marillier"
)

var appName = "KrankyBear ThreatInvaders"
var appCopyright = "Copyright (c) Allan Marillier, 2025-" + strconv.Itoa(time.Now().Year())

func init() {
	// Detect executable name to determine app name
	execPath := os.Args[0]
	execName := filepath.Base(execPath)
	// Remove extension if present (e.g., .exe on Windows)
	execNameWithoutExt := strings.TrimSuffix(execName, filepath.Ext(execName))
	if strings.Contains(strings.ToLower(execNameWithoutExt), "tanium") {
		appName = "KrankyBear Tanium Threat Invaders"
	} else {
		appName = "KrankyBear ThreatInvaders"
	}
}

func main() {
	myApp := app.NewWithID("com.github.amarillier.KrankyBearThreatInvaders")
	myApp.SetIcon(resourceKrankyBearUSArmyCombatHelmetWoodlandPng)
	myApp.Settings().SetTheme(newAppTheme())
	
	ui := NewGameUI(myApp)
	ui.Show()
	
	myApp.Run()
}

// "Now this is not the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
