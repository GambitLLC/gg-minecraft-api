package main

import (
	"bed.gg/minecraft-api/v2/src/config"
	"bed.gg/minecraft-api/v2/src/logger"
	"bed.gg/profile-scanner/v2/scanner"
	"context"
	"github.com/go-redis/redis/v9"
	"go.uber.org/zap"
	"net"
	"time"

	"bed.gg/minecraft-api/v2/src/api"
	"github.com/meilisearch/meilisearch-go"
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

	// -- call the scanner --
	scanner.Scanner(handler, index)
}
