// Copyright 2019-2020 Rajesh Malviya
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/natspubsub"
)

func main() {
	numTopics := flag.Int("topics", 100, "Number of topics")
	interval := flag.Duration("interval", time.Second, "Time interval")
	flag.Parse()

	log.SetFlags(log.Lshortfile)

	ctx := context.Background()

	natsConn, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalln("Error while connecting to nats server: ", err)
	}

	// add a signal notifier to close the streamer gracefully
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill)

	// Open N topics via the nats driver.
	var topics = make([]*pubsub.Topic, *numTopics)

	go func() {
		log.Println("sig:", <-sigs)
		log.Println("Gracefully closing the server")

		for _, topic := range topics {
			topic.Shutdown(ctx)
		}

		natsConn.Close()

		os.Exit(0)
	}()

	for i := 0; i < *numTopics; i++ {
		topic, err := natspubsub.OpenTopic(natsConn, "topic"+strconv.Itoa(i), nil)
		if err != nil {
			log.Fatal(err)
		}
		topics[i] = topic
	}

	log.Printf("Opened %v topics\n", *numTopics)
	log.Printf("Sending message every %vs\n", interval.Seconds())

	for {
		log.Println("Sending message")

		var wg sync.WaitGroup

		for j := 0; j < *numTopics; j++ {
			wg.Add(1)
			go func(i int) {
				eventString := fmt.Sprintf("Topic%v %v", i, time.Now())

				d, _ := json.Marshal(map[string]string{"message": eventString})

				err := topics[i].Send(ctx, &pubsub.Message{Body: d})
				if err != nil {
					log.Fatal(err)
				}
				wg.Done()
			}(j)
		}
		time.Sleep(*interval)
		wg.Wait()
	}

}
