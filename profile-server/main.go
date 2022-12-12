package main

import (
	"context"
	"net"

	"bed.gg/minecraft-api/v2/src/api"
	"bed.gg/minecraft-api/v2/src/config"
	"bed.gg/minecraft-api/v2/src/logger"
	"github.com/go-redis/redis/v9"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go.uber.org/zap"
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
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// -- setup the ip config --
	var ipPool []net.IP

	for _, ip := range config.Ip.Pool {
		ipPool = append(ipPool, net.ParseIP(ip))
	}

	// -- create the api handler --
	handler := &api.Handler{
		Logger: lg,
		Rdb:    rdb,
		Ctx:    context.Background(),
		IPPool: ipPool,
		IpIdx:  0,
	}

	// -- fiber app --
	app := fiber.New()

	//setup cors
	app.Use(cors.New(cors.Config{
		AllowOrigins: "https://bed.gg, https://www.bed.gg, http://localhost:3000, http://127.0.0.1:3000",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	// -- register routes --
	app.Get("/profile/:uuid", handler.GetProfile)
	app.Get("/profiles", handler.GetProfiles)
	app.Get("/texture/:textureid", handler.GetTexture)
	app.Get("/textures", handler.GetTextures)

	// -- start the server --
	lg.Fatal("%s", app.Listen(":8080"))
}
