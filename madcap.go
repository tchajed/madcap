package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func loadSongs(rootpath string) []Song {
	songs := make([]Song, 0)
	songQueue := make(chan Song)
	workQueue := make(chan string)
	songsProcessed := 0
	go func() {
		filepath.Walk(rootpath, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() || songsProcessed > 50 {
				return nil
			}
			workQueue <- path
			songsProcessed++
			return nil
		})
		close(workQueue)
		return
	}()
	done := make(chan bool)
	workersRemaining := 4
	for i := 0; i < workersRemaining; i++ {
		go func() {
			for {
				songpath, ok := <-workQueue
				// if workQueue is empty then exit
				if !ok {
					done <- true
					return
				}
				song, err := loadSong(songpath)
				if err != nil {
					continue
				}
				songQueue <- song
			}
		}()
	}
	for workersRemaining > 0 {
		select {
		case song := <-songQueue:
			songs = append(songs, song)
		case <-done:
			workersRemaining--
		}
	}
	return songs
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "not enough arguments")
		os.Exit(1)
	}
	rootdir := flag.Args()[0]
	songs := loadSongs(rootdir)
	for _, song := range songs {
		fmt.Println(song)
	}
}
