package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"bed.gg/minecraft-api/v2/src/api"
	"encoding/base64"
	"github.com/bep/debounce"
	"github.com/gofiber/fiber/v2"
	"github.com/meilisearch/meilisearch-go"
)

type TextureResponse struct {
	Timestamp         int64  `json:"timestamp"`
	ProfileId         string `json:"profileId"`
	ProfileName       string `json:"profileName"`
	SignatureRequired bool   `json:"signatureRequired"`
	Textures          struct {
		Skin SkinURL `json:"SKIN"`
		Cape CapeURL `json:"CAPE"`
	} `json:"textures"`
}

type SkinURL struct {
	Url      string `json:"url"`
	Metadata struct {
		Model string `json:"model"`
	} `json:"metadata"`
}

type CapeURL struct {
	Url string `json:"url"`
}

type Document struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Textures Textures `json:"textures"`
}

type Textures struct {
	Skin Skin `json:"skin,omitempty"`
	Cape Cape `json:"cape,omitempty"`
}

type Skin struct {
	Data string `json:"data"`
}

type Cape struct {
	Data string `json:"data"`
}

func main() {
	//meilisearch client
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host: "http://127.0.0.1:7700",
	})

	//set the index
	index := client.Index("players")

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
		"9032ea59-caa1-4489-a167-c19a32f9771d",
	}

	apiHandler := api.Handler{}

	limit := make(chan struct{}, 2)

	//temporary initial sleep time for hao's machine
	sleepTime := int32(50)

	debounced := debounce.New(time.Second)

	increment := func() {
		sleepTime++
		fmt.Printf("incremented sleepTime: %d\n", sleepTime)
	}

	wg := &sync.WaitGroup{}

	for _, id := range uuidList {
		wg.Add(1)

		limit <- struct{}{}

		go func(id string) {
			defer wg.Done()

			//fetch the profile based on the uuid from mojang
			code, profile, _, _ := apiHandler.FetchProfile(id)
			if code == fiber.StatusTooManyRequests {
				fmt.Println("429 received")

				//after debounce
				debounced(increment)

			} else {
				//TODO: Requeue for lookup again
				//fmt.Println(code, sleepTime)
			}

			//parse textures from profile.Properties
			texturesDataBase64String := profile.Properties[0].Value
			textureDataJsonString, _ := base64.StdEncoding.DecodeString(texturesDataBase64String)

			textureResponse := &TextureResponse{}
			_ = json.Unmarshal(textureDataJsonString, textureResponse)

			skinResponse := ""
			capeResponse := ""

			//fetch the textures from mojang
			if textureResponse.Textures.Skin.Url != "" {
				splitString := strings.Split(textureResponse.Textures.Skin.Url, "/")
				textureid := splitString[len(splitString)-1]
				_, skinResponse, _, _ = apiHandler.FetchTexture(textureid)
			}

			if textureResponse.Textures.Cape.Url != "" {
				splitString := strings.Split(textureResponse.Textures.Cape.Url, "/")
				textureid := splitString[len(splitString)-1]
				_, capeResponse, _, _ = apiHandler.FetchTexture(textureid)
			}

			//populate doc with Textures
			doc := Document{
				Id:   profile.Id,
				Name: profile.Name,
				Textures: Textures{
					Skin: Skin{
						Data: skinResponse,
					},
					Cape: Cape{
						Data: capeResponse,
					},
				},
			}

			//check if the doc differs from what is in meilisearch
			searchRes, _ := index.Search(profile.Id, &meilisearch.SearchRequest{
				Limit: 1,
			})

			if len(searchRes.Hits) == 1 {
				//document exists, check for differences before updating meilisearch
				searchJsonString, _ := json.Marshal(searchRes.Hits[0])
				foundDoc := &Document{}
				_ = json.Unmarshal(searchJsonString, foundDoc)

				if doc.Name != foundDoc.Name || doc.Textures.Skin.Data != foundDoc.Textures.Skin.Data || doc.Textures.Cape.Data != foundDoc.Textures.Cape.Data {
					//updating doc to meilisearch, doc data differs from foundDoc
					var docs = [1]Document{doc}

					task, err := index.AddDocuments(docs)

					if err != nil {
						fmt.Println(err)
						os.Exit(1)
					}

					fmt.Printf("Updating doc: %d\n", task.TaskUID)
				}

			} else {
				//adding doc to meilisearch, doc does not exist
				var docs = [1]Document{doc}

				task, err := index.AddDocuments(docs)

				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}

				fmt.Printf("Creating doc: %d\n", task.TaskUID)
			}

			<-limit
		}(id)

		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}

	wg.Wait()

}
