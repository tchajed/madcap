package main

import (
	"bytes"
	"fmt"
	"github.com/ascherkus/go-id3/src/id3"
	"madcap/spectral"
	"os"
	"os/exec"
	"strconv"
)

type Song struct {
	Features []float64
	Info     map[string]string
}

func (s Song) Vector() []float64 {
	return s.Features
}

func (s Song) String() string {
	_, hasTitle := s.Info["title"]
	if hasTitle {
		return fmt.Sprintf("%s - %s", s.Info["artist"], s.Info["title"])
	}
	return s.Info["filename"]
}

type Songs []Song

func (s Songs) Vector(i int) []float64 {
	return s[i].Features
}

func (s Songs) Dim() int {
	if len(s) == 0 {
		return 0
	}
	return len(s[0].Features)
}

func (s Songs) Len() int {
	return len(s)
}

var soxFormat []string = []string{"-r", "22.05k", "-e", "signed", "-b", "8", "-c", "1", "-t", ".raw"}

// construct the sox effect to trim to get a specific start/length sample
// expects start and length in seconds
func soxTrim(start int, length int) []string {
	return []string{"trim", strconv.Itoa(start), strconv.Itoa(length)}
}

func loadSong(filename string) (song Song, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	song.Info = make(map[string]string)
	id3info := id3.Read(f)
	if id3info != nil {
		song.Info["title"] = id3info.Name
		song.Info["artist"] = id3info.Artist
	}
	song.Info["filename"] = filename
	f.Close()
	arguments := []string{filename}
	arguments = append(arguments, soxFormat...)
	arguments = append(arguments, "-")
	arguments = append(arguments, soxTrim(30, 4)...)
	cmd := exec.Command("sox", arguments...)
	buffer := new(bytes.Buffer)
	cmd.Stdout = buffer
	err = cmd.Run()
	if err != nil {
		return Song{}, err
	}
	samples := make([]float64, buffer.Len())
	for i, sample := range buffer.Bytes() {
		samples[i] = float64(sample)
	}
	spectrogram := spectral.Compute(samples, 1024, 0.75)
	statistics := spectrogram.Stats(22050)
	features := []string{"cutoffFreq", "energyCV", "maxEnergyFreq", "maxVarFreq"}
	song.Features = make([]float64, 0, len(features))
	for _, stat := range features {
		song.Features = append(song.Features, statistics[stat])
	}
	song.Features = append(song.Features, spectrogram.LogFreq...)
	return
}
