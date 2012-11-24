package main

import (
	"flag"
	"fmt"
	"github.com/tchajed/madcap/cluster"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"
)

// Load a directory of songs recursively, using a function to determine whether
// or not to process a path. This can be used, eg, to sample the space in some
// way.
func loadSongs(rootpath string, processSong func(path string, num int) bool) []Song {
	songs := make([]Song, 0)
	// queue of paths as found by the traversal
	pathQueue := make(chan string)
	// queue of songs as loaded by the workers
	songQueue := make(chan Song)
	songNum := 1
	go func() {
		filepath.Walk(rootpath, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if !processSong(path, songNum) {
				return nil
			}
			pathQueue <- path
			songNum++
			if songNum%25 == 0 {
				fmt.Fprintln(os.Stderr, songNum)
			}
			return nil
		})
		close(pathQueue)
		return
	}()
	done := make(chan bool)
	// this is the max number of simultaneous calls to loadSong
	workersRemaining := 8
	for i := 0; i < workersRemaining; i++ {
		go func() {
			for {
				songpath, ok := <-pathQueue
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
	// pull all the songs out of the songQueue, waiting for the workers to finish
	// and load them into a slice
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
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	var songLimit = flag.Int("limit", 0, "song limit (<= 0 means no limit)")
	var songFrac = flag.Float64("frac", 1, "fraction of songs to consider")
	var octavefile = flag.String("octave", "", "file to output octave data to")
	var jsfile = flag.String("js", "", "file to output js data to")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "not enough arguments")
		os.Exit(1)
	}
	// Write out a cpuprofile; runtime/pprof makes this ridiculously easy
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	// makes logic simpler
	if *songLimit <= 0 {
		*songLimit = math.MaxInt32
	}
	rootdir := flag.Args()[0]
	rand.Seed(time.Now().UnixNano())
	songs := loadSongs(rootdir, func(path string, num int) bool {
		return rand.Float64() < *songFrac && num < *songLimit
	})
	/*
		// Trim the feature lists to 2 dimensions (a very poor college student's
		// "dimensionality reduction")
		// TODO: implement bindings to LAPACK or implement a basic eigendecomposition
		for _, song := range songs {
			song.Features = song.Features[0:1]
		}
	*/
	k := 4
	assignments, cost := cluster.Kmeans(Songs(songs), k, cluster.EuclideanDistance)
	for i := 0; i < 20; i++ {
		newassignments, newcost := cluster.Kmeans(Songs(songs), k, cluster.EuclideanDistance)
		if newcost < cost {
			assignments, cost = newassignments, newcost
		}
	}
	groups := make([][]Song, k)
	for i, cluster := range assignments {
		groups[cluster] = append(groups[cluster], songs[i])
	}
	// octave/matlab output
	if *octavefile != "" {
		f, err := os.Create(*octavefile)
		if err != nil {
			log.Fatal(err.Error())
		}
		for groupi, group := range groups {
			fmt.Fprintln(f, fmt.Sprintf("groups%d = [", groupi))
			fmt.Fprintln(os.Stderr, groupi, len(group))
			for _, song := range group {
				for _, feat := range song.Features {
					fmt.Fprintf(f, "%f ", feat)
				}
				fmt.Fprintf(f, ";\n")
			}
			fmt.Fprintln(f, "];")
		}
	}
	if *jsfile != "" {
		f, err := os.Create(*jsfile)
		if err != nil {
			log.Fatal(err.Error())
		}
		fmt.Fprintln(f, "songs = [")
		for groupi, group := range groups {
			for _, song := range group {
				fmt.Fprintf(f, "{")
				// output features
				fmt.Fprintf(f, "\"features\":[")
				for _, v := range song.Features {
					fmt.Fprintf(f, "%f,\n", v)
				}
				fmt.Fprintf(f, "],")
				// output cluster label
				fmt.Fprintf(f, "\"cluster\":%d,", groupi)
				fmt.Fprintf(f, "\"info\":\"%s\",", song.String())
				fmt.Fprintf(f, "},\n")
			}
		}
		fmt.Fprintln(f, "];")
	}
}
