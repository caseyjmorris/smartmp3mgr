package main

import (
	"encoding/json"
	"fmt"
	"github.com/caseyjmorris/smartmp3mgr/internal"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:  smartmp3mgr (sum|record|find-new) (args)")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "sum":
		for _, file := range os.Args[2:] {
			mp3, err := internal.ParseMP3(file)
			if err != nil {
				fmt.Println(err)
				fmt.Println("usage:  smartmp3mgr sum [files]")
				os.Exit(1)
			}
			fmt.Printf("%q:  %s\r\n", file, mp3.Hash)
		}
		os.Exit(0)
	}

	fmt.Println("Hello world")
	fmt.Println(os.Args[1])

	files, err := internal.FindMP3Files(os.Args[1])

	if err != nil {
		fmt.Printf("%q", err)
		os.Exit(1)
	}

	for _, file := range files {
		parsed, err := internal.ParseMP3(file)
		if err != nil {
			fmt.Println(err)
		} else {
			complet, _ := json.Marshal(parsed)
			fmt.Println(string(complet))
		}
	}
}
