package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
	return nil
}

//FIXME: some bug with the caching and
// getting: Get "": unsupported protocol scheme ""

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
		// fmt.Println("in cache")

		// convert the bytes to a struct
		err := json.Unmarshal(locationAreasBytes, &locationAreas)
		if err != nil {
			return err
		}
	} else {
		// fmt.Println("not in cache")

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
		// fmt.Printf("cached url=[%s] with the contents [%v]", url, locationAreasBytes)

		// fmt.Printf("cache entries: %v\n", cache.entries)
	}

	// print the names of the 20 location areas
	for _, locationArea := range locationAreas.Results {
		fmt.Println(locationArea.Name)
	}

	// update the mapConfig next and previous fields
	mapConfig.Next = &locationAreas.Next
	mapConfig.Previous = &locationAreas.Previous

	// fmt.Println("next: ", *mapConfig.Next)
	// fmt.Println("previous: ", *mapConfig.Previous)

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
	// fmt.Printf("url: %s\n", url)

	cache := args[1].(*Cache)
	var locationAreas LocationAreas

	//  check if the url to search is in the cache
	locationAreasBytes, ok := cache.Get(url)

	if ok {
		// fmt.Println("in cache")

		// convert the bytes to a struct
		err := json.Unmarshal(locationAreasBytes, &locationAreas)
		if err != nil {
			return err
		}

	} else {
		// fmt.Println("not in cache")

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

	// REPL loop
	for {
		fmt.Print("pokedex > ")
		// wait for user input
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		cmd := input.Text()

		// try except
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
