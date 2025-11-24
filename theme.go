//go:generate fyne bundle -o bundled.go -a Resources/Images/KrankyBearUSArmyCombatHelmetWoodland.png Resources/Images/tanium.png Resources/Images/KrankyBearHardHat.png Resources/Sounds/wheeHoo.mp3 Resources/Sounds/zip.mp3 Resources/Sounds/boing.mp3 Resources/Sounds/explosion.mp3

package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type appTheme struct {
	base fyne.Theme
}

func newAppTheme() fyne.Theme {
	return &appTheme{
		base: theme.DefaultTheme(),
	}
}

func (a *appTheme) Size(n fyne.ThemeSizeName) float32 {
	if n == theme.SizeNameHeadingText {
		return a.base.Size(n) * 1.5
	}

	return a.base.Size(n)
}

func (a *appTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return a.base.Color(n, v)
}

func (a *appTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return a.base.Icon(n)
}

func (a *appTheme) Font(style fyne.TextStyle) fyne.Resource {
	return a.base.Font(style)
}

// "Now this is not the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
