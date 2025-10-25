package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type cliCommand struct {
	name        string
	description string
	callback    func() error
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
	}
}

func CleanInput(text string) []string {
	str := strings.ToLower(text)
	trimmed := strings.TrimSpace(str)
	return strings.Split(trimmed, " ")
}

func commandExit() error {
	fmt.Print("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp() error {
	fmt.Printf("Welcome to the Pokedex!\n")
	commands := getCommands()
	for _, c := range commands {
		fmt.Printf(" %s: %s\n", c.name, c.description)
	}
	return nil
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	commands := getCommands()

	for {
		fmt.Print("Pokedex > ")
		scanner.Scan()
		str := scanner.Text()
		words := CleanInput(str)
		if len(words) == 0 {
			continue
		}
		commandName := words[0]

		if command, exists := commands[commandName]; exists {
			command.callback()
		} else {
			fmt.Printf("unknown command: %s\n", commandName)
		}
	}
}
