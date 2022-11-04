package main

import (
	"bed.gg/minecraft-api/v2/logger"
	"context"
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"os"
	"regexp"
	"strings"
	"time"
)

var ctx = context.Background()

type ApiHandler struct {
	Logger *logger.ZapLogger
	rdb    *redis.Client
}

func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func isValidUsername(u string) bool {
	matched, err := regexp.Match("^[a-zA-Z0-9_]{2,16}$", []byte(u))

	if err != nil {
		return false
	}

	return matched
}

func (h *ApiHandler) fetchProfile(playerUUID string) (int, []byte, []error) {
	//fetch the profile from mojang api
	a := fiber.AcquireAgent()
	req := a.Request()
	req.Header.SetMethod(fiber.MethodGet)
	req.SetRequestURI(fmt.Sprintf("https://sessionserver.mojang.com/session/minecraft/profile/%s", playerUUID))

	if err := a.Parse(); err != nil {
		h.Logger.Error("%v", err)
		return fiber.StatusInternalServerError, nil, []error{err}
	}

	return a.Bytes()
}

func (h *ApiHandler) getProfile(c *fiber.Ctx) error {
	playerUUID := c.Params("uuid")
	remoteAddr := c.Context().Conn().RemoteAddr().String()

	h.Logger.Info("%s GET /profile/%s", remoteAddr, playerUUID)

	if isValidUUID(playerUUID) {
		//check if the profile exists in redis
		item, err := h.rdb.Get(ctx, playerUUID).Result()
		if err == redis.Nil {
			h.Logger.Info("[%s] Cache Miss for %s", playerUUID, remoteAddr)
			code, profile, errs := h.fetchProfile(playerUUID)

			if len(errs) > 0 {
				for _, err := range errs {
					if err != nil {
						h.Logger.Error("%v", err)
					}
				}
				return c.SendStatus(code)
			}

			err = h.rdb.Set(ctx, playerUUID, profile, 0).Err()

			if err != nil {
				h.Logger.Error("[%s] Failed to set data: %v", playerUUID, err)
			} else {
				err = h.rdb.Set(ctx, fmt.Sprintf("%s:exp", playerUUID), "", 15*time.Second).Err()

				if err != nil {
					h.Logger.Error("[%s] Failed to set expiration: %v", playerUUID, err)
				}
			}

			c.Status(code)
			return c.SendString(string(profile))
		}

		if err != nil {
			h.Logger.Error("%v", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		h.Logger.Info("[%s] Cache Hit for [%s]", playerUUID, c.Context().Conn().RemoteAddr().String())
		c.Status(fiber.StatusOK)
		return c.SendString(item)

	} else {
		c.Status(fiber.StatusBadRequest)
		return c.SendString(fmt.Sprintf("bad uuid: %s", playerUUID))
	}
}

func (h *ApiHandler) getUUID(c *fiber.Ctx) error {
	username := c.Params("username")

	if isValidUsername(username) {
		//fetch the uuid from mojang api
		a := fiber.AcquireAgent()
		req := a.Request()
		req.Header.SetMethod(fiber.MethodGet)
		req.SetRequestURI(fmt.Sprintf("https://api.mojang.com/users/profiles/minecraft/%s", username))

		if err := a.Parse(); err != nil {
			h.Logger.Error("%v", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		code, body, errs := a.Bytes()

		if len(errs) > 0 {
			for _, err := range errs {
				if err != nil {
					h.Logger.Error("%v", err)
				}
			}
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		c.Status(code)
		return c.SendString(string(body))
	} else {
		c.Status(fiber.StatusBadRequest)
		return c.SendString(fmt.Sprintf("bad username: %s", username))
	}
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
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// -- create the api handler --
	apiHandler := &ApiHandler{
		Logger: lg,
		rdb:    rdb,
	}

	// -- configure keyspace events --
	err := rdb.ConfigSet(ctx, "notify-keyspace-events", "Ex").Err()

	if err != nil {
		fmt.Printf("unable to set keyspace events %v", err.Error())
		os.Exit(1)
	}

	// -- subscribe to keyspace events --
	pubsub := rdb.PSubscribe(ctx, "__key*__:*")

	//-- profile ttl listener --
	go func(h *ApiHandler, pubsub *redis.PubSub) {
		channel := pubsub.Channel()

		for msg := range channel {
			playerUUID := strings.Split(msg.Payload, ":")[0]

			// fetch new data from api
			code, profile, errs := h.fetchProfile(playerUUID)

			if len(errs) > 0 {
				for _, err := range errs {
					if err != nil {
						h.Logger.Error("%v", err)
					}
				}
				continue
			}

			if code != fiber.StatusOK {
				h.Logger.Info("[%s] Returned Code: %d", playerUUID, code)
			}

			err = h.rdb.Set(ctx, playerUUID, profile, 0).Err()

			if err != nil {
				h.Logger.Error("[%s] Failed to set data: %v", playerUUID, err)
			} else {
				err = h.rdb.Set(ctx, fmt.Sprintf("%s:exp", playerUUID), "", 15*time.Second).Err()

				if err != nil {
					h.Logger.Error("[%s] Failed to set expiration: %v", playerUUID, err)
				}
			}

			// write into redis

			// update the ttl

		}
	}(pubsub)

	// -- fiber app --
	app := fiber.New()

	app.Get("/profile/:uuid", apiHandler.getProfile)
	app.Get("/uuid/:username", apiHandler.getUUID)

	lg.Fatal("%s", app.Listen(":8080"))
}
