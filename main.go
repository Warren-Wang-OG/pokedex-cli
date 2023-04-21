package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type cacheEntry struct {
	createdAt time.Time
	val       []byte
}

// NOTE: maps are NOT thread safe
func (cache) add(key string, val []byte) {
	// TODO:
}

func (cache) get(key string) ([]byte, bool) {
	// TODO:
	return nil, false
}

func (cache) reapleap() {
	// TODO:
}

type Command struct {
	name        string
	description string
	callback    Callback
}

type Callback interface {
	Execute(args ...interface{}) error
}

type NoParamFunc func() error
type ParamFunc func(args ...interface{}) error

func (f NoParamFunc) Execute(args ...interface{}) error {
	return f()
}

func (f ParamFunc) Execute(args ...interface{}) error {
	return f(args...)
}

type LocationAreas struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
		Url  string `json:"url"`
	} `json:"results"`
}

type MapConfig struct {
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
}

func helpCommand() error {
	fmt.Println("This is the Pokemon Pokedex CLI")
	fmt.Println("Available commands:")
	fmt.Println("help - Show help (display this msg)")
	fmt.Println("exit - Exit the CLI")
	fmt.Println("map - Displays the names of the next 20 location areas")
	fmt.Println("mapb - Displays the names of the previous 20 location areas")
	return nil
}

// use pokedex API to get the names of 20 location areas
// make a GET request to the API here: https://pokeapi.co/api/v2/location-area/
// and print the names of the 20 location areas
func mapCommand(args ...interface{}) error {
	mapConfig := args[0].(*MapConfig)
	url := *mapConfig.Next
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// decode the response body into a struct
	var locationAreas LocationAreas
	err = json.NewDecoder(resp.Body).Decode(&locationAreas)
	if err != nil {
		return err
	}

	// print the names of the 20 location areas
	for _, locationArea := range locationAreas.Results {
		fmt.Println(locationArea.Name)
	}

	// update the mapConfig next and previous fields
	mapConfig.Next = &locationAreas.Next
	mapConfig.Previous = &locationAreas.Previous

	return nil
}

// get the names of the previous 20 location areas
func mapbCommand(args ...interface{}) error {
	mapConfig := args[0].(*MapConfig)
	// if no previous page, return an error
	if mapConfig.Previous == nil || *mapConfig.Previous == "" {
		return fmt.Errorf("no previous page")
	}
	url := *mapConfig.Previous

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// decode the response body into a struct
	var locationAreas LocationAreas
	err = json.NewDecoder(resp.Body).Decode(&locationAreas)
	if err != nil {
		return err
	}

	// print the names of the 20 location areas
	for _, locationArea := range locationAreas.Results {
		fmt.Println(locationArea.Name)
	}

	// update the mapConfig next and previous fields
	mapConfig.Next = &locationAreas.Next
	mapConfig.Previous = &locationAreas.Previous

	return nil
}

func main() {
	// map from command name to command
	cmdHandler := make(map[string]Command)
	cmdHandler["help"] = Command{
		name:        "help",
		description: "Show help",
		callback:    NoParamFunc(helpCommand),
	}

	cmdHandler["exit"] = Command{
		name:        "exit",
		description: "Exit the CLI",
		callback:    NoParamFunc(func() error { os.Exit(0); return nil }),
	}

	initMapURL := "https://pokeapi.co/api/v2/location-area/"
	mapConfig := MapConfig{
		Next:     &initMapURL,
		Previous: nil,
	}

	cmdHandler["map"] = Command{
		name:        "map",
		description: "Displays the names of the next 20 location areas",
		callback:    ParamFunc(mapCommand),
	}

	cmdHandler["mapb"] = Command{
		name:        "map",
		description: "Displays the names of the previous 20 location areas",
		callback:    ParamFunc(mapbCommand),
	}

	// cache for maps
	cache := make(map[string]cacheEntry)

	// REPL loop
	for {
		fmt.Print("pokedex > ")
		// wait for user input
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		cmd := input.Text()

		// try except
		if cmd == "map" || cmd == "mapb" {
			err := cmdHandler[cmd].callback.Execute(&mapConfig)
			if err != nil {
				fmt.Println(err)
			}
		} else if cmdHandler[cmd].callback != nil {
			cmdHandler[cmd].callback.Execute()
		} else {
			fmt.Println("Command not found")
		}
	}
}
