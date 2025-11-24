package main

import (
	"bytes"
	"io"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
)

// SoundManager manages sound effects
type SoundManager struct {
	enabled      bool
	sampleRate   beep.SampleRate
	wheeHooData  []byte
	zipData      []byte
	boingData    []byte
	explosionData []byte
}

// NewSoundManager creates a new sound manager
func NewSoundManager(enabled bool) (*SoundManager, error) {
	sr := beep.SampleRate(44100)
	
	// Initialize speaker
	err := speaker.Init(sr, sr.N(time.Second/10))
	if err != nil {
		return nil, err
	}
	
	sm := &SoundManager{
		enabled:       enabled,
		sampleRate:    sr,
		wheeHooData:   resourceWheeHooMp3Data,
		zipData:       resourceZipMp3Data,
		boingData:     resourceBoingMp3Data,
		explosionData: resourceExplosionMp3Data,
	}
	
	return sm, nil
}

// playSound plays a sound from byte data
func (sm *SoundManager) playSound(data []byte) {
	if !sm.enabled {
		return
	}
	
	go func() {
		reader := bytes.NewReader(data)
		stream, format, err := mp3.Decode(io.NopCloser(reader))
		if err != nil {
			return
		}
		defer stream.Close()
		
		resampled := beep.Resample(4, format.SampleRate, sm.sampleRate, stream)
		speaker.Play(beep.Seq(resampled, beep.Callback(func() {})))
	}()
}

// SetEnabled enables or disables sound effects
func (sm *SoundManager) SetEnabled(enabled bool) {
	sm.enabled = enabled
}

// IsEnabled returns whether sound effects are enabled
func (sm *SoundManager) IsEnabled() bool {
	return sm.enabled
}

// PlayWheeHoo plays the level complete sound
func (sm *SoundManager) PlayWheeHoo() {
	sm.playSound(sm.wheeHooData)
}

// PlayZip plays the zip sound (for hitting invaders)
func (sm *SoundManager) PlayZip() {
	sm.playSound(sm.zipData)
}

// PlayBoing plays the boing sound (for hitting invaders)
func (sm *SoundManager) PlayBoing() {
	sm.playSound(sm.boingData)
}

// PlayExplosion plays the explosion sound (for losing a life)
func (sm *SoundManager) PlayExplosion() {
	sm.playSound(sm.explosionData)
}

// Close closes the sound manager and releases resources
func (sm *SoundManager) Close() {
	// Resources are managed by beep/speaker
}

