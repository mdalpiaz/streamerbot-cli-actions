package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/eiannone/keyboard"
)

type SentAction struct {
	Action struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"action"`
}

type ReceivedActions struct {
	Count   int `json:"count"`
	Actions []struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		Group           string `json:"group"`
		Enabled         bool   `json:"enabled"`
		SubactionsCount int    `json:"subactions_count"`
	} `json:"actions"`
}

func main() {
	resp, err := http.Get("http://127.0.0.1:7474/GetActions")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var actions ReceivedActions
	if err := json.Unmarshal([]byte(body), &actions); err != nil {
		return
	}

	keysEvents, err := keyboard.GetKeys(10)
	if err != nil {
		panic(err)
	}
	defer func() { keyboard.Close() }()

	fmt.Println("Press ESC to quit")
	for {
		event := <-keysEvents
		if event.Err != nil || event.Key == keyboard.KeyEsc {
			break
		}
		fmt.Printf("You Pressed: %c, %X\n", event.Rune, event.Key)
	}
}
