package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"runtime"

	"golang.org/x/net/html"
)

func main() {
	// optional - get the number of concurrent routines from the user.
	routineCount := *flag.Int("routines", runtime.NumCPU(),
		"Number of concurrent goroutines for a specific task. If not specified, number of logical CPUs will be used as a baseline")
	flag.Parse()

	// initialize word bank consisting of valid words
	wordBankReady := make(chan struct{}, 1)
	var wb map[string]struct{}
	var err error
	go func() {
		wb, err = wordBank()
		if err != nil {
			log.Fatalf("could not create wordBank: %s", err)
		}
		wordBankReady <- struct{}{}
	}()

	// parse file containing URLs
	urls, err := urlBank()
	if err != nil {
		log.Fatalf("unable to populate url bank: %s", err)
	}

	// get htmls
	htmls := make(chan *html.Node, routineCount*5)
	go retrieveHTMLEssays(urls, htmls)
	// wait until the word bank is ready
	<-wordBankReady

	// parse html and count words in every article and return to intermediate maps
	parser := &htmlParser{
		wordbank:   wb,
		routines:   routineCount,
		htmlStream: htmls,
	}

	intermediateMaps, err := parser.parseAndCount()
	if err != nil {
		log.Fatalf("unable to parse and count essays: %w", err)
	}

	// unite maps
	message := "Top 10 words that occurred the most in the provided articles:\n"
	result := TenMostOccurred(intermediateMaps)
	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err == nil {
		fmt.Println(message + string(jsonResult))
		return
	}
	log.Printf("Warning: unable to display result in json form: %s", err)
	fmt.Print(message)
	for key, value := range result {
		fmt.Printf("%s: %d\n", key, value)
	}

}
