package main

import (
	"bed.gg/minecraft-api/v2/src/config"
	"bed.gg/minecraft-api/v2/src/logger"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v9"
	"go.uber.org/zap"
	"net"
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
	// -- create a new logger --
	lg := logger.NewLogger()

	//-- defer flushing writes --
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			panic(err)
		}
	}(lg.Logger)

	// -- connect to redis --
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis-store:6380",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// -- setup the ip config --
	var ipPool []net.IP

	for _, ip := range config.Ip.Pool {
		lg.Info("Registered IP: %s", ip)
		ipPool = append(ipPool, net.ParseIP(ip))
	}

	// -- meilisearch client --
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   "http://meilisearch:7700",
		APIKey: "RIGHT_PARENTHESIS-ubr-Auc-NINE",
	})

	// -- create the api handler --
	handler := api.Handler{
		Logger:   lg,
		Rdb:      rdb,
		MSClient: client,
		Ctx:      context.Background(),
	}

	// -- create the index --
	index := client.Index("players")
	_, err := client.CreateIndex(&meilisearch.IndexConfig{
		Uid: "players",
	})

	if err != nil {
		handler.Logger.Error("%v", err)
		return
	}

	// -- wait for meilisearch to initialize --
	time.Sleep(5 * time.Second)

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

	limit := make(chan struct{}, 2)

	//temporary initial sleep time for hao's machine
	sleepTime := int32(50)

	debounced := debounce.New(time.Second)

	increment := func() {
		sleepTime++
		handler.Logger.Info("incremented sleepTime: %d", sleepTime)
	}

	wg := &sync.WaitGroup{}

	for {
		for _, id := range uuidList {
			wg.Add(1)

			limit <- struct{}{}

			go func(handler api.Handler, wg *sync.WaitGroup, id string) {
				defer func() {
					<-limit
					wg.Done()
				}()

				//fetch the profile based on the uuid from mojang
				code, profile, _, errs := handler.FetchProfile(id)

				if len(errs) != 0 {
					//TODO: Requeue for lookup again
					for _, err := range errs {
						handler.Logger.Error("%v", err)
					}

					return
				}

				if code == fiber.StatusTooManyRequests {
					handler.Logger.Error("429 received!")

					//after debounce
					debounced(increment)

					return
				}

				if code == fiber.StatusOK {
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
						_, skinResponse, _, errs = handler.FetchTexture(textureid)

						if len(errs) != 0 {
							//TODO: Requeue for lookup again
							for _, err := range errs {
								handler.Logger.Error("%v", err)
							}

							return
						}
					}

					if textureResponse.Textures.Cape.Url != "" {
						splitString := strings.Split(textureResponse.Textures.Cape.Url, "/")
						textureid := splitString[len(splitString)-1]
						_, capeResponse, _, errs = handler.FetchTexture(textureid)

						if len(errs) != 0 {
							//TODO: Requeue for lookup again
							for _, err := range errs {
								handler.Logger.Error("%v", err)
							}

							return
						}
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

					//check if the doc differs from redis
					exists, item, _ := handler.CacheGet(fmt.Sprintf("scanner:%s", doc.Id))

					if !exists {
						//item does not exist in cache, put into cache and meilisearch
						docJsonString, _ := json.Marshal(&doc)
						err = handler.CachePut(fmt.Sprintf("scanner:%s", doc.Id), string(docJsonString), 0)

						if err != nil {
							handler.Logger.Error("%v", err)
							return
						}

						//adding doc to meilisearch
						var docs = [1]Document{doc}

						task, err := index.AddDocuments(docs)

						if err != nil {
							handler.Logger.Error("%v", err)
							return
						}

						handler.Logger.Info("Creating doc (%s): %d", doc.Id, task.TaskUID)

					} else {
						//item exists in cache, check if differs and then put in meilisearch
						foundDoc := &Document{}
						_ = json.Unmarshal([]byte(item), foundDoc)

						if doc.Name != foundDoc.Name || doc.Textures.Skin.Data != foundDoc.Textures.Skin.Data || doc.Textures.Cape.Data != foundDoc.Textures.Cape.Data {
							//updating doc to cache and meilisearch, doc data differs from foundDoc
							docJsonString, _ := json.Marshal(&doc)
							err = handler.CachePut(fmt.Sprintf("scanner:%s", doc.Id), string(docJsonString), 0)

							if err != nil {
								handler.Logger.Error("%v", err)
								return
							}

							var docs = [1]Document{doc}

							task, err := index.AddDocuments(docs)

							if err != nil {
								handler.Logger.Error("%v", err)
								return
							}
							handler.Logger.Info("Updating doc (%s): %d", doc.Id, task.TaskUID)
						}
					}
				}
			}(handler, wg, id)

			time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		}
	}
}
