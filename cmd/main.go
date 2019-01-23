package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/miguelbernadi/monkeylearn-go/pkg/monkeylearn"
)

func main() {
	token := flag.String("token", "", "Monkeylearn token")
	classifier := flag.String("classifier", "", "Monkeylearn classifier ID")
	rpm := flag.Int64("rpm", 120, "Requests per minute (should be lower than API rate limit)")
	batchSize := flag.Int("batch", 1, "Documents per batch")
	flag.Parse()

	if *token == "" {
		log.Fatal("Token is mandatory")
	}

	docs := []string{
		"aabb",
		"bbaa",
	}
	fmt.Printf("Documents to classify: %d\n", len(docs))

	fmt.Printf("Batch size: %d\n", *batchSize)
	batches := monkeylearn.SplitInBatches(docs, *batchSize)
	fmt.Printf("Number of batches: %d\n", len(batches))

	client := monkeylearn.NewClient(*token)
	count := 0
	for resp := range loop(time.Minute / time.Duration(*rpm), batches, client, *classifier) {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil { log.Panic(err) }
		log.Printf("%#v\n", string(body))
		count++
	}
	log.Printf("Processed batch %d out of %d", count, len(batches))
}

func loop(rate time.Duration, batches []*monkeylearn.Batch, client *monkeylearn.Client, classifier string) (out chan *http.Response) {
	out = make(chan *http.Response)

	throttle := time.Tick(rate)
	var wg sync.WaitGroup
	for _, batch := range batches {
		wg.Add(1)
		<-throttle  // rate limit
		go func(batch *monkeylearn.Batch) {
			out <- client.Classify(classifier, batch)
			wg.Done()
		}(batch)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
