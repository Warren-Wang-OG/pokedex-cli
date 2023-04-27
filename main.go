package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Cache struct {
	entries map[string]cacheEntry
	mutex   sync.Mutex
}

type cacheEntry struct {
	createdAt time.Time
	val       []byte
}

// create and return a new cache
func NewCache(interval time.Duration) *Cache {
	cache := Cache{
		entries: make(map[string]cacheEntry),
	}

	// run the old cache cleaner in a goroutine
	go cache.Reaploop(interval)

	return &cache
}

// add a new (key, value) pair to the cache
func (cache *Cache) Add(key string, val []byte) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.entries[key] = cacheEntry{
		createdAt: time.Now(),
		val:       val,
	}
}

// (key, value) = (url to query, response body)
// returns the value and a boolean indicating if the key was found
func (cache *Cache) Get(key string) ([]byte, bool) {
	// use locks to make map access thread safe
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	val, ok := cache.entries[key]

	if ok {
		return val.val, true
	}
	return nil, false
}

// called whenever NewCache is called
// each time an interval passes, remove all entries in the cache that are older than the interval
func (cache *Cache) Reaploop(interval time.Duration) {
	for {
		time.Sleep(interval)

		cache.mutex.Lock()

		// list of keys to delete
		toDelete := []string{}

		for key, val := range cache.entries {
			if time.Since(val.createdAt) > interval {
				toDelete = append(toDelete, key)
			}
		}

		for _, key := range toDelete {
			delete(cache.entries, key)
		}

		cache.mutex.Unlock()
	}
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
	fmt.Println("explore [location] - show all pokemon in a location")
	return nil
}

// use pokedex API to get the names of 20 location areas
// and print the names of the 20 location areas
func mapCommand(args ...interface{}) error {
	mapConfig := args[0].(*MapConfig)
	cache := args[1].(*Cache)
	var locationAreas LocationAreas
	url := *mapConfig.Next

	//  check if the url to search is in the cache
	locationAreasBytes, ok := cache.Get(url)

	if ok {
		// convert the bytes to a struct
		err := json.Unmarshal(locationAreasBytes, &locationAreas)
		if err != nil {
			return err
		}
	} else {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// decode the response body into a struct
		err = json.NewDecoder(resp.Body).Decode(&locationAreas)
		if err != nil {
			return err
		}

		// cache the response body
		// convert the struct to bytes
		locationAreasBytes, err := json.Marshal(locationAreas)
		if err != nil {
			return err
		}
		// save the bytes in the cache
		cache.Add(url, locationAreasBytes)
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
	cache := args[1].(*Cache)
	var locationAreas LocationAreas

	//  check if the url to search is in the cache
	locationAreasBytes, ok := cache.Get(url)

	if ok {
		// convert the bytes to a struct
		err := json.Unmarshal(locationAreasBytes, &locationAreas)
		if err != nil {
			return err
		}

	} else {
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

		// cache the response body
		// convert the struct to bytes
		locationAreasBytes, err := json.Marshal(locationAreas)
		if err != nil {
			return err
		}
		// save the bytes in the cache
		cache.Add(url, locationAreasBytes) // cache the response body
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

type ExploreRequest struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Location struct {
		Name string `json:"name"`
	} `json:"location_area"`
	Pokemon_encounters []struct {
		Pokemon struct {
			Id   int    `json:"id"`
			Name string `json:"name"`
		} `json:"pokemon"`
		VersionDetails []struct {
			Rate int `json:"rate"`
		} `json:"version_details"`
	} `json:"pokemon_encounters"`
}

// show all pokemon in a location
func exploreCommand(args ...interface{}) error {
	location := args[0].(string)
	cache := args[1].(*Cache)
	location_url := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%s", location)
	var exploreRequest ExploreRequest

	// check if the location is in the cache
	exploreRequestBytes, ok := cache.Get(location)
	if ok {
		// convert the bytes to a struct
		err := json.Unmarshal(exploreRequestBytes, &exploreRequest)
		if err != nil {
			return err
		}
	} else {
		resp, err := http.Get(location_url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// decode the response body into a struct
		err = json.NewDecoder(resp.Body).Decode(&exploreRequest)
		if err != nil {
			return err
		}

		// save in cache
		// convert the struct to bytes
		exploreRequestBytes, err := json.Marshal(exploreRequest)
		if err != nil {
			return err
		}
		cache.Add(location, exploreRequestBytes)
	}

	// print the pokemon
	fmt.Println("Exploring", exploreRequest.Name)
	fmt.Println("Pokemon encounters:")
	for _, pokemon := range exploreRequest.Pokemon_encounters {
		fmt.Println("-", pokemon.Pokemon.Name)
	}

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

	// initialize the mapConfig and initial url starting
	initMapURL := "https://pokeapi.co/api/v2/location-area/?offset=0&limit=20"
	mapConfig := MapConfig{
		Next:     &initMapURL,
		Previous: nil,
	}
	// cache for maps add a reasonable interval like 5 minutes
	var cache *Cache = NewCache(5 * time.Minute)

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

	cmdHandler["explore"] = Command{
		name:        "explore",
		description: "show all pokemon in a location",
		callback:    ParamFunc(exploreCommand),
	}

	// REPL loop
	for {
		fmt.Print("pokedex > ")
		// wait for user input
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		cmd := input.Text()
		if cmd == "" {
			continue
		}
		params := strings.Split(cmd, " ")

		// ---------- explore ----------
		if len(params) == 2 {
			if params[0] != "explore" {
				fmt.Println("Command not found")
				continue
			}

			err := cmdHandler[params[0]].callback.Execute(params[1], cache)
			if err != nil {
				fmt.Println(err)
			}
			continue
		}
		// ---------- explore ----------

		if cmd == "explore" {
			fmt.Println("Please enter a location")
			continue
		}

		if cmd == "map" || cmd == "mapb" {
			err := cmdHandler[cmd].callback.Execute(&mapConfig, cache)
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
