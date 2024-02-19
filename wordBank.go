package main

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"
)

const (
	wordBankPath  = "https://raw.githubusercontent.com/dwyl/english-words/master/words.txt"
	minWordLength = 3
)

var errUnavailableBank = errors.New("unable to retrieve raw word bank from provided url")

// wordBank reads raw bank from a given url, filters it, and returns as an array
func wordBank() (map[string]struct{}, error) {
	// the assumption here is that the link contains no duplicates
	resp, err := http.Get(wordBankPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errUnavailableBank, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w. response code: %s", errUnavailableBank, resp.Status)
	}
	scanner := bufio.NewScanner(resp.Body)
	// A map is more suitable than an array since we don't need to iterate over it when searching a key.
	validWords := make(map[string]struct{})
	for scanner.Scan() {
		word := scanner.Text()
		// Check if the word satisfies the conditions
		if len(word) >= minWordLength && isAlphabetic(word) {
			// save everything in lower case so because 'Apple' and 'apple' are considered the same word.
			validWords[strings.ToLower(word)] = struct{}{}
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed scanning raw word bank: %w", err)
	}

	return validWords, nil

}

// isAlphabetic checks if a string contains only alphabetic characters
func isAlphabetic(s string) bool {
	for _, char := range s {
		if !unicode.IsLetter(char) {
			return false
		}
	}
	return true
}
