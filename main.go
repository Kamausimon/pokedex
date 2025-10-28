package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Kamausimon/pokedex/internal"
)

type cliCommand struct {
	name        string
	description string
	callback    func(*Config, ...string) error
}

type Config struct {
	Next     string
	Previous string
	Cache    *internal.Cache
}

func getCommands() map[string]cliCommand {
	return map[string]cliCommand{
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    commandExit,
		},
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback:    commandHelp,
		},

		"map": {
			name:        "map",
			description: "Get all location areas",
			callback:    commandMap,
		},
		"mapb": {
			name:        "mapb",
			description: "get all previous location areas",
			callback:    commandMapB,
		},
		"explore": {
			name:        "explore",
			description: "explores the highlighted areas",
			callback:    commandExplore,
		},
	}
}

func CleanInput(text string) []string {
	str := strings.ToLower(text)
	trimmed := strings.TrimSpace(str)
	return strings.Split(trimmed, " ")
}

func commandExit(cfg *Config, args ...string) error {
	fmt.Print("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(cfg *Config, args ...string) error {
	fmt.Printf("Welcome to the Pokedex!\n")
	commands := getCommands()
	for _, c := range commands {
		fmt.Printf(" %s: %s\n", c.name, c.description)
	}
	return nil
}

func fetchWithCache(url string, cache *internal.Cache) ([]byte, error) {
	if cachedData, found := cache.Get(url); found {
		return cachedData, nil
	}

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode > 299 {
		return nil, fmt.Errorf("request failed with status code %d", res.StatusCode)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	cache.Add(url, data)
	return data, nil
}

func commandExplore(cfg *Config, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("you must provide a name or id")
	}
	locationName := args[0]
	url := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%s", locationName)

	result, err := fetchWithCache(url, cfg.Cache)
	if err != nil {
		return err
	}

	var locationResponse struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Location struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"location"`
		PokemonEncounters []struct {
			Pokemon struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"pokemon"`
			VersionDetails []struct {
				EncounterDetails []struct {
					Chance   int `json:"chance"`
					MaxLevel int `json:"max_level"`
					MinLevel int `json:"min_level"`
				} `json:"encounter_details"`
			} `json:"version_details"`
		} `json:"pokemon_encounters"`
	}

	err = json.Unmarshal(result, &locationResponse)
	if err != nil {
		return fmt.Errorf("error parsing location data %w", err)
	}

	fmt.Printf("exploring %s", locationResponse.Name)
	fmt.Println("found pokemon:")

	if len(locationResponse.PokemonEncounters) == 0 {
		fmt.Println("No pokemon found in this area")
		return nil
	}

	for _, encounter := range locationResponse.PokemonEncounters {
		fmt.Printf("- %s\n", encounter.Pokemon.Name)
	}

	return nil
}

func commandMap(cfg *Config, args ...string) error {
	url := "https://pokeapi.co/api/v2/location-area/"
	if cfg.Next != "" {
		url = cfg.Next
	}
	result, err := fetchWithCache(url, cfg.Cache)
	if err != nil {
		return err
	}

	var apiResponse struct {
		Next     *string `json:"next"`
		Previous *string `json:"previous"`
		Results  []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"results"`
	}

	err = json.Unmarshal(result, &apiResponse)
	if err != nil {
		return err
	}

	if apiResponse.Next != nil {
		cfg.Next = *apiResponse.Next
	} else {
		cfg.Next = ""
	}

	if apiResponse.Previous != nil {
		cfg.Previous = *apiResponse.Previous
	} else {
		cfg.Previous = ""
	}

	for _, area := range apiResponse.Results {
		fmt.Println(area.Name)
	}

	return nil
}

func commandMapB(cfg *Config, args ...string) error {
	if cfg.Previous == "" {
		fmt.Println("You're on the first page")
		return nil
	}
	result, err := fetchWithCache(cfg.Previous, cfg.Cache)
	if err != nil {
		return err
	}

	var apiResponse struct {
		Next     *string `json:"next"`
		Previous *string `json:"previous"`
		Results  []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"results"`
	}

	err = json.Unmarshal(result, &apiResponse)
	if err != nil {
		return err
	}

	if apiResponse.Next != nil {
		cfg.Next = *apiResponse.Next
	} else {
		cfg.Next = ""
	}

	if apiResponse.Previous != nil {
		cfg.Previous = *apiResponse.Previous
	} else {
		cfg.Previous = ""
	}

	for _, area := range apiResponse.Results {
		fmt.Println(area.Name)
	}

	return nil
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	commands := getCommands()
	cfg := &Config{
		Cache: internal.NewCache(5 * time.Minute),
	}

	for {
		fmt.Print("Pokedex > ")
		scanner.Scan()
		str := scanner.Text()
		words := CleanInput(str)
		if len(words) == 0 {
			continue
		}
		commandName := words[0]
		args := []string{}
		if len(words) > 1 {
			args = words[1:]
		}

		if command, exists := commands[commandName]; exists {
			err := command.callback(cfg, args...)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		} else {
			fmt.Printf("unknown command: %s\n", commandName)
		}
	}
}
