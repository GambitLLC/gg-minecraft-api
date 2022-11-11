package main

import (
	"fmt"
	"time"

	"bed.gg/minecraft-api/v2/src/api"
	"github.com/bep/debounce"
	"github.com/gofiber/fiber/v2"
)

type Document struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Textures Textures `json:"textures"`
}

type Textures struct {
	Skin *Skin `json:"skin,omitempty"`
	Cape *Cape `json:"cape,omitempty"`
}

type Skin struct {
	Data  string `json:"data"`
	Model string `json:"model,omitempty"`
}

type Cape struct {
	Data string `json:"data"`
}

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

	//how to fetch texture from mojang api
	code, base64Texture, body, errs := apiHandler.FetchTexture("b0cc08840700447322d953a02b965f1d65a13a603bf64b17c803c21446fe1635")
	_ = code
	_ = base64Texture
	_ = body
	_ = errs

	limit := make(chan struct{}, 2)

	//temporary initial sleep time for hao's machine
	sleepTime := int32(50)

	debounced := debounce.New(time.Second)

	increment := func() {
		sleepTime++
		fmt.Printf("incremented sleepTime: %d\n", sleepTime)
	}

	for {
		for _, id := range uuidList {

			limit <- struct{}{}

			go func(id string) {
				code, profile, _, errs := apiHandler.FetchProfile(id)
				if code == fiber.StatusTooManyRequests {
					fmt.Println("429 received")

					//after debounce
					debounced(increment)

				} else {
					fmt.Println(code, sleepTime)
				}

				// TODO: handle errs
				_ = errs

				// TODO: parse textures from profile.Properties
				_ = profile

				// TODO: populate doc with Textures
				doc := Document{
					Id:   profile.Id,
					Name: profile.Name,
					// Textures: Textures{},
				}

				// TODO: check if received doc differs from what is saved in db
				_ = doc

				<-limit
			}(id)

			time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		}
	}
}
