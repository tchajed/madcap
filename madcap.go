package main

import (
	"bytes"
	"flag"
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
	id3info := id3.Read(f)
	if id3info != nil {
		song.Info = make(map[string]string)
		song.Info["title"] = id3info.Name
		song.Info["artist"] = id3info.Artist
	}
	f.Close()
	arguments := []string{filename}
	arguments = append(arguments, soxFormat...)
	arguments = append(arguments, "-")
	arguments = append(arguments, soxTrim(0, 5)...)
	cmd := exec.Command("sox", arguments...)
	buffer := new(bytes.Buffer)
	cmd.Stdout = buffer
	err = cmd.Run()
	if err != nil {
		fmt.Println(err.Error())
	}
	samples := make([]float64, buffer.Len())
	for i, sample := range buffer.Bytes() {
		samples[i] = float64(sample)
	}
	spectrogram := spectral.Compute(samples, 1024, 0.75)
	statistics := spectrogram.Stats(22050)
	features := []string{"cutoffFreq", "energyCV", "maxVarFreq", "maxEnergyFreq"}
	song.Features = make([]float64, 0, len(features))
	for _, stat := range features {
		song.Features = append(song.Features, statistics[stat])
	}
	return
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "not enough arguments")
		os.Exit(1)
	}
	filename := flag.Args()[0]
	song, _ := loadSong(filename)
	fmt.Println(song)
}
