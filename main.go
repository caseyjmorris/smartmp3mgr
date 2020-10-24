package main

import (
	"encoding/json"
	"fmt"
	"github.com/caseyjmorris/smartmp3mgr/internal"
	"os"
)

func main() {
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
