package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Command struct {
	name        string
	description string
	callback    func() error
}

type CommandWithParam struct {
	name        string
	description string
	callback    func(config *mapConfig) error
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

type mapConfig struct {
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
func mapCommand(config mapConfig) error {
	url := "https://pokeapi.co/api/v2/location-area/"
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

	return nil
}

func mapbCommand() error {
	// TODO: use pokedex API to get the names of 20 location areas
	return nil
}

func main() {
	// map from command name to command
	cmdHandler := make(map[string]Command)
	cmdHandler["help"] = Command{
		name:        "help",
		description: "Show help",
		callback:    helpCommand,
	}

	cmdHandler["exit"] = Command{
		name:        "exit",
		description: "Exit the CLI",
		callback:    func() error { os.Exit(0); return nil },
	}

	mapConfig := mapConfig{
		Next:     nil,
		Previous: nil,
	}

	cmdHandler["map"] = Command{
		name:        "map",
		description: "Displays the names of the next 20 location areas",
		callback:    mapCommand,
	}

	cmdHandler["mapb"] = Command{
		name:        "map",
		description: "Displays the names of the previous 20 location areas",
		callback:    mapbCommand,
	}

	// REPL loop
	for {
		fmt.Print("pokedex > ")
		// wait for user input
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		cmd := input.Text()

		switch cmd {
		case "map":
			mapCommand(&mapConfig)
		case "mapb":
			mapbCommand(&mapConfig)
		default:
			// try except
			if cmdHandler[cmd].callback != nil {
				cmdHandler[cmd].callback()
			} else {
				fmt.Println("Command not found")
			}
		}

	}
}
