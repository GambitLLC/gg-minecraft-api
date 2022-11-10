package main

import (
	"fmt"
	"time"

	"bed.gg/minecraft-api/v2/src/api"
	"github.com/bep/debounce"
	"github.com/gofiber/fiber/v2"
)

func main() {
	//construct a list of uuids
	uuidList := []string{
		"e71be459-ee50-4ec8-93dd-0dfce4a5efd6",
		"5ea79bfa-f22d-420a-8505-69ba87443966",
		"fc12d601-05c3-4622-b737-bd292e4c8bff",
		"340e98bb-4f20-45bd-bef3-08ce502bb01c",
		"7ed9b051-20ef-4222-a89b-9ff8c3f03a52",
		"b781c162-0673-4b0a-b904-c18dc8052531",
		"5003956c-a84b-4a6e-a8c0-5e1a7ca89856",
		"4c40404c-4f3e-4bf4-b937-b2602e61bddf",
		"a977b265-6e80-4bd2-be45-be58084b1f8d",
		"f7c77d99-9f15-4a66-a87d-c4a51ef30d19",
	}

	apiHandler := api.Handler{}

	limit := make(chan struct{}, 2)
	sleepTime := int32(19)

	debounced := debounce.New(time.Second)

	increment := func() {
		sleepTime++
		fmt.Printf("incremented sleepTime: %d\n", sleepTime)
	}

	for {
		for _, id := range uuidList {

			limit <- struct{}{}

			go func(id string) {
				code, _, _, _ := apiHandler.FetchProfile(id)
				if code == fiber.StatusTooManyRequests {
					fmt.Println("429 recieved")

					//after debounce
					debounced(increment)

				} else {
					fmt.Println(code, sleepTime)
				}

				<-limit
			}(id)

			time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		}
	}
}
