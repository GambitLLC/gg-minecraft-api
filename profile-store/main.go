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

const API_KEY = "RIGHT_BRACE-Gret-Hok-cech-ONE"

type StoreHandler struct {
	api.Handler
}

func (h *StoreHandler) GetSignIn(c *fiber.Ctx) error {
	h.Logger.Info("%v", c.GetReqHeaders())

	apiKey := c.GetReqHeaders()["X-Bedgg-Api-Key"]

	if apiKey != API_KEY {
		h.Logger.Error("Incorrect API Key provided: %s", apiKey)
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	playerUUID := c.Params("uuid")
	remoteAddr := c.Context().Conn().RemoteAddr().String()

	h.Logger.Info("%s POST /signIn/%s", remoteAddr, playerUUID)

	if api.IsValidUUID(playerUUID) {
		//publish the new signup
		err := h.Handler.Rdb.Publish(h.Handler.Ctx, "signIn", playerUUID).Err()

		if err != nil {
			h.Logger.Error("SignUp Error: %v", err)
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
		Addr:     "redis-store:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// -- create the api handler --
	handler := NewStoreHandler(lg, rdb)

	// -- fiber app --
	app := fiber.New()

	// -- register routes --
	app.Get("/signIn/:uuid", handler.GetSignIn)

	// -- start the server --
	lg.Fatal("%s", app.Listen(":8080"))
}
