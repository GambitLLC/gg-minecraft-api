package main

import (
	"bed.gg/minecraft-api/v2/src/logger"
	"context"
	"fmt"

	"bed.gg/minecraft-api/v2/src/api"

	"github.com/go-redis/redis/v9"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type StoreHandler struct {
	api.Handler
}

func (h *StoreHandler) PostSignUp(c *fiber.Ctx) error {
	playerUUID := c.Params("uuid")
	remoteAddr := c.Context().Conn().RemoteAddr().String()

	h.Logger.Info("%s POST /signUp/%s", remoteAddr, playerUUID)

	if api.IsValidUUID(playerUUID) {
		//cache the UUID in the redis
		err := h.Handler.CachePut(fmt.Sprintf("store:%s", playerUUID), "", 0)

		if err != nil {
			h.Logger.Error("%v", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.SendStatus(fiber.StatusOK)
	} else {
		c.Status(fiber.StatusBadRequest)
		return c.SendString(fmt.Sprintf("bad uuid: %s", playerUUID))
	}
}

func NewStoreHandler(lg *logger.ZapLogger, rdb *redis.Client) *StoreHandler {
	handler := &StoreHandler{}

	handler.Logger = lg
	handler.Rdb = rdb
	handler.Ctx = context.Background()
	handler.IpIdx = 0

	return handler
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

	// -- create the api handler --
	handler := NewStoreHandler(lg, rdb)

	// -- fiber app --
	app := fiber.New()

	// -- register routes --
	app.Post("/signUp/:uuid", handler.PostSignUp)

	// -- start the server --
	lg.Fatal("%s", app.Listen(":8080"))
}
