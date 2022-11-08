package main

import (
	"bed.gg/minecraft-api/v2/api"
	"bed.gg/minecraft-api/v2/logger"
	"context"
	"github.com/go-redis/redis/v9"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"net"
)

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
	handler := &api.Handler{
		Logger: lg,
		Rdb:    rdb,
		Ctx:    context.Background(),
		IPPool: []net.IP{
			net.ParseIP("10.0.0.4"),
			net.ParseIP("10.0.0.5"),
			net.ParseIP("10.0.0.6"),
			net.ParseIP("10.0.0.7"),
			net.ParseIP("10.0.0.8"),
		},
		IpIdx: 0,
	}

	// -- fiber app --
	app := fiber.New()

	// -- register routes --
	app.Get("/profile/:uuid", handler.GetProfile)
	//app.Get("/uuid/:username", handler.GetUUID)

	// -- start the server --
	lg.Fatal("%s", app.Listen(":8080"))
}
