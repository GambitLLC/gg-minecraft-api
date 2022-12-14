package scanner

import (
	"bed.gg/minecraft-api/v2/src/api"
	"bed.gg/profile-scanner/v2/mojang"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/bep/debounce"
	"github.com/go-redis/redis/v9"
	"github.com/gofiber/fiber/v2"
	"github.com/meilisearch/meilisearch-go"
	"strings"
	"time"
)

type UUIDPool struct {
	PriorityJobs chan string
	Jobs         chan string
}

func NewUUIDPool() *UUIDPool {
	return &UUIDPool{
		PriorityJobs: make(chan string, 4096),
		Jobs:         make(chan string, 128),
	}
}

func handleJob(handler api.Handler, index *meilisearch.Index, limit chan struct{}, debounced func(f func()), increment func(), uuid string) {
	defer func() {
		<-limit
	}()

	//fetch the profile based on the uuid from mojang
	code, profile, _, errs := handler.FetchProfile(uuid)

	//rate limited by mojang api, slow down request speed
	if code == fiber.StatusTooManyRequests {
		handler.Logger.Error("429 received!")

		//after debounce
		debounced(increment)

		return
	}

	//other errors occured while fetching profile
	if len(errs) != 0 {
		//TODO: Requeue for lookup again
		for _, err := range errs {
			handler.Logger.Error("%v", err)
		}

		return
	}

	//successfully fetched profile from mojang
	if code == fiber.StatusOK {
		//parse textures from profile.Properties
		texturesDataBase64String := profile.Properties[0].Value
		textureDataJsonString, _ := base64.StdEncoding.DecodeString(texturesDataBase64String)

		textureResponse := &mojang.TextureResponse{}
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
		doc := mojang.Document{
			Id:   profile.Id,
			Name: profile.Name,
			Textures: mojang.Textures{
				Skin: mojang.Skin{
					Data: skinResponse,
				},
				Cape: mojang.Cape{
					Data: capeResponse,
				},
			},
		}

		//check if the doc differs from redis
		exists, item, _ := handler.CacheGet(fmt.Sprintf("scanner:%s", doc.Id))

		if !exists {
			//item does not exist in cache, put into cache and meilisearch
			docJsonString, _ := json.Marshal(&doc)
			err := handler.CachePut(fmt.Sprintf("scanner:%s", doc.Id), string(docJsonString), 0)

			if err != nil {
				handler.Logger.Error("%v", err)
				return
			}

			//adding doc to meilisearch
			var docs = [1]mojang.Document{doc}

			task, err := index.AddDocuments(docs)

			if err != nil {
				handler.Logger.Error("%v", err)
				return
			}

			handler.Logger.Info("Creating doc (%s): %d", doc.Id, task.TaskUID)

		} else {
			//item exists in cache, check if differs and then put in meilisearch
			foundDoc := &mojang.Document{}
			_ = json.Unmarshal([]byte(item), foundDoc)

			if doc.Name != foundDoc.Name || doc.Textures.Skin.Data != foundDoc.Textures.Skin.Data || doc.Textures.Cape.Data != foundDoc.Textures.Cape.Data {
				//updating doc to cache and meilisearch, doc data differs from foundDoc
				docJsonString, _ := json.Marshal(&doc)
				err := handler.CachePut(fmt.Sprintf("scanner:%s", doc.Id), string(docJsonString), 0)

				if err != nil {
					handler.Logger.Error("%v", err)
					return
				}

				var docs = [1]mojang.Document{doc}

				task, err := index.AddDocuments(docs)

				if err != nil {
					handler.Logger.Error("%v", err)
					return
				}
				handler.Logger.Info("Updating doc (%s): %d", doc.Id, task.TaskUID)
			}
		}
	}
}

func Scanner(h api.Handler, index *meilisearch.Index) {
	uuidPool := NewUUIDPool()
	limit := make(chan struct{}, 2)

	sleepTime := int32(50)

	debounced := debounce.New(time.Second)

	increment := func() {
		sleepTime++
		h.Logger.Info("incremented sleepTime: %d", sleepTime)
	}

	//populate priority jobs with new sign-ups
	pubsub := h.Rdb.Subscribe(h.Ctx, "signIn")
	go func(ctx context.Context, pubsub *redis.PubSub, uuidPool *UUIDPool) {
		ch := pubsub.Channel()

		for msg := range ch {
			uuidPool.PriorityJobs <- msg.Payload
		}
	}(h.Ctx, pubsub, uuidPool)

	//populate non-priority jobs
	go func(handler api.Handler, uuidPool *UUIDPool) {
		pointer := uint64(0)

		for {
			keys, cursor, err := handler.Rdb.Scan(h.Ctx, pointer, "scanner:*", 128).Result()

			if err != nil {
				h.Logger.Error("Scan Error: %v", err)
				continue
			}

			for _, key := range keys {
				uuid := strings.Split(key, ":")[1]
				uuidPool.Jobs <- uuid
			}

			pointer = cursor
		}
	}(h, uuidPool)

	//scanner main loops
	for {
		limit <- struct{}{}

		select {
		case priorityUUID := <-uuidPool.PriorityJobs:
			//handle priority jobs first
			h.Logger.Info("Priority Job: %s", priorityUUID)
			go handleJob(h, index, limit, debounced, increment, priorityUUID)
		default:
			//handle whichever job comes first
			select {
			case priorityUUID := <-uuidPool.PriorityJobs:
				//handle priority job
				h.Logger.Info("Priority Job: %s", priorityUUID)
				go handleJob(h, index, limit, debounced, increment, priorityUUID)
			case uuid := <-uuidPool.Jobs:
				//handle non-priority job
				h.Logger.Info("Job: %s", uuid)
				go handleJob(h, index, limit, debounced, increment, uuid)
			}
		}

		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
}
