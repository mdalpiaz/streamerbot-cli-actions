package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/eiannone/keyboard"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Optional[T any] struct {
	Value  T
	Exists bool
}

type Args struct {
	ip   Optional[string]
	port Optional[int]
}

type SimpleAction struct {
	id   string
	name string
}

type DoAction struct {
	Action struct {
		ID string `json:"id"`
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

type MenuState int

const (
	StateMenu = iota
	StateAdd
	StateRemove
	StateMain
)

func ToOptional[T any](value T) Optional[T] {
	return Optional[T]{value, true}
}

func ReadNumber(reader *bufio.Reader) (int, error) {
	text, err := reader.ReadString('\n')
	if err != nil {
		return 0, err
	}
	text = strings.ReplaceAll(text, "\n", "")
	text = strings.ReplaceAll(text, "\r", "")
	number, err := strconv.Atoi(text)
	if err != nil {
		return 0, err
	}
	return number, nil
}

func GetActions(url string) ReceivedActions {
	resp, err := http.Get(fmt.Sprintf("%sGetActions", url))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var availableActions ReceivedActions
	err = json.NewDecoder(resp.Body).Decode(&availableActions)
	if err != nil {
		panic(err)
	}
	return availableActions
}

func main() {
	connectionArgs := Args{
		ip:   Optional[string]{Exists: false},
		port: Optional[int]{Exists: false},
	}
	args := os.Args[1:]

	for i := 0; i+1 < len(args); i++ {
		switch args[i] {
		case "-a":
			i++
			connectionArgs.ip = ToOptional(args[i])
		case "-p":
			i++
			port, err := strconv.Atoi(args[i])
			if err != nil {
				fmt.Println("Port is not a number")
				return
			}
			connectionArgs.port = ToOptional(port)
		}
	}

	reader := bufio.NewReader(os.Stdin)

	if !connectionArgs.ip.Exists {
		fmt.Println("Enter Streamer.Bot Server IP:")
		text, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		text = strings.ReplaceAll(text, "\n", "")
		text = strings.ReplaceAll(text, "\r", "")
		connectionArgs.ip = ToOptional(text)
	}
	if !connectionArgs.port.Exists {
		fmt.Println("Enter Streamer.Bot Server Port:")
		port, err := ReadNumber(reader)
		if err != nil {
			panic(err)
		}
		connectionArgs.port = ToOptional(port)
	}

	url := fmt.Sprintf("http://%s:%d/", connectionArgs.ip.Value, connectionArgs.port.Value)

	err := keyboard.Open()
	if err != nil {
		panic(err)
	}
	defer func() { _ = keyboard.Close() }()

	state := StateMenu
	actionsMap := make(map[rune]SimpleAction)

	fmt.Printf("Press ESC to quit\n")
	for {
		if state == StateMenu {
			fmt.Printf("a...Add\n")
			fmt.Printf("r...Remove\n")
			fmt.Printf("m...Macros\n")

			rune, key, err := keyboard.GetKey()
			if err != nil {
				panic(err)
			}
			if key == keyboard.KeyEsc {
				break
			} else if rune == 'a' {
				state = StateAdd
			} else if rune == 'r' {
				state = StateRemove
			} else if rune == 'm' {
				state = StateMain
				fmt.Printf("Macro mode active\n")
			}
		} else if state == StateAdd {
			fmt.Printf("Available Actions:\n")
			actions := GetActions(url)
			for i, a := range actions.Actions {
				fmt.Printf("%d) %s\n", i, a.Name)
			}
			_ = keyboard.Close()
			fmt.Printf("Enter index: ")
			index, err := ReadNumber(reader)
			if err != nil {
				continue
			}
			a := actions.Actions[index]
			_ = keyboard.Open()
			fmt.Printf("Enter Key to assign: ")
			rune, _, err := keyboard.GetKey()
			if err != nil {
				panic(err)
			}
			fmt.Printf("%c\n", rune)
			actionsMap[rune] = SimpleAction{a.ID, a.Name}
			state = StateMenu
		} else if state == StateRemove {
			for r, a := range actionsMap {
				fmt.Printf("%c -> %s\n", r, a.name)
			}
			fmt.Printf("Enter key of mapping to remove: ")
			rune, _, err := keyboard.GetKey()
			if err != nil {
				panic(err)
			}
			fmt.Printf("%c\n", rune)
			delete(actionsMap, rune)
			state = StateMenu
		} else if state == StateMain {
			rune, key, err := keyboard.GetKey()
			if err != nil {
				panic(err)
			}
			if key == keyboard.KeyEsc {
				state = StateMenu
				continue
			}

			fmt.Printf("Pressed: %c, %d\n", rune, key)
			ac, ok := actionsMap[rune]
			if !ok {
				continue
			}
			action := DoAction{}
			action.Action.ID = ac.id
			data, err := json.Marshal(action)
			if err != nil {
				panic(err)
			}
			resp, err := http.Post(fmt.Sprintf("%sDoAction", url), "application/json", bytes.NewBuffer(data))
			if err != nil || resp.StatusCode != 204 {
				fmt.Printf("Error while sending request\n")
				continue
			}
			defer resp.Body.Close()
		}
	}
	fmt.Println("Exiting")
	os.Exit(0)
}
