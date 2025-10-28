package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
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
	Next          string
	Previous      string
	Cache         *internal.Cache
	CurrentArea   string
	CaughtPokemon map[string]bool
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
		"catch": {
			name:        "catch",
			description: "catches a pokemon",
			callback:    commandCatch,
		},
		"inspect": {
			name:        "inspect",
			description: "see details about a pokemon",
			callback:    commandInspect,
		},
		"pokedex": {
			name:        "pokedex",
			description: "view caught pokemon",
			callback:    commandPokedex,
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

func commandPokedex(cfg *Config, args ...string) error {
	if len(cfg.CaughtPokemon) == 0 {
		fmt.Println("You are yet to catch any pokemon")
		return nil
	}
	fmt.Println("Your pokemon:")
	for pokemon := range cfg.CaughtPokemon {
		fmt.Printf("- %s\n", pokemon)
	}
	return nil
}

func commandExplore(cfg *Config, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("you must provide a name or id")
	}
	locationName := args[0]
	url := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%s/", locationName)

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

	fmt.Printf("exploring %s\n", locationResponse.Name)
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

func commandCatch(cfg *Config, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("please include the pokemon name")
	}

	pokemonName := args[0]
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s", pokemonName)

	result, err := fetchWithCache(url, cfg.Cache)
	if err != nil {
		return fmt.Errorf("there was an error getting the result: %s", err)
	}

	var pokemonResponse struct {
		ID             int    `json:"id"`
		Name           string `json:"name"`
		BaseExperience int    `json:"base_experience"`
	}

	err = json.Unmarshal(result, &pokemonResponse)
	if err != nil {
		return fmt.Errorf("there was an error parsing the pokemon data: %s", err)
	}

	fmt.Printf("catch %s\n", pokemonResponse.Name)
	fmt.Printf("Throwing a Pokeball at %s...\n", pokemonResponse.Name)
	if attemptCatch(pokemonResponse.BaseExperience) {
		fmt.Printf("%s was caught!\n", pokemonResponse.Name)
		if cfg.CaughtPokemon == nil {
			cfg.CaughtPokemon = make(map[string]bool)
		}
		cfg.CaughtPokemon[pokemonResponse.Name] = true
	} else {
		fmt.Printf("%s escaped!\n", pokemonResponse.Name)
	}

	return nil
}

func attemptCatch(baseExperience int) bool {
	shakeValue := 65536 / (255 / calculateCatchRate(baseExperience))

	for i := 0; i < 4; i++ {
		roll := rand.Intn(65536)
		if roll >= shakeValue {
			fmt.Printf("broke out after %d shakes\n", i)
			return false
		}
		fmt.Print("shake...")
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println("\nCaught!")
	return true
}

func calculateCatchRate(baseExp int) int {
	rate := 255 - (baseExp / 3)
	if rate < 3 {
		rate = 3
	}
	return rate
}

func commandInspect(cfg *Config, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("please provide a name for the pokemon")
	}
	pokemonName := args[0]
	if cfg.CaughtPokemon[pokemonName] {
		return fmt.Errorf("you have not caught that pokemon")
	}
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s/", pokemonName)
	result, err := fetchWithCache(url, cfg.Cache)
	if err != nil {
		return err
	}
	var pokemon struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Height int    `json:"height"`
		Weight string `json:"weight"`
		Stats  []struct {
			BaseStat int `json:"base_stat"`
			Stat     struct {
				Name string `json:"name"`
			} `json:"stat"`
		} `json:"stats"`
		Types []struct {
			Type struct {
				Name string `json:"name"`
			} `json:"type"`
		} `json:"types"`
	}

	err = json.Unmarshal(result, &pokemon)
	if err != nil {
		return err
	}
	// Display Pokemon info
	fmt.Printf("Name: %s\n", pokemon.Name)
	fmt.Printf("Height: %d\n", pokemon.Height)
	fmt.Printf("Weight: %s\n", pokemon.Weight)

	fmt.Println("Stats:")
	for _, stat := range pokemon.Stats {
		fmt.Printf("  -%s: %d\n", stat.Stat.Name, stat.BaseStat)
	}

	fmt.Println("Types:")
	for _, typeInfo := range pokemon.Types {
		fmt.Printf("  - %s\n", typeInfo.Type.Name)
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
		Cache:         internal.NewCache(5 * time.Minute),
		CaughtPokemon: make(map[string]bool),
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
