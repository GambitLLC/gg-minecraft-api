package api

import (
	"bed.gg/minecraft-api/v2/logger"
	"context"
	"github.com/go-redis/redis/v9"
	"github.com/gofiber/fiber/v2"
	"net"
	"sync"
	"testing"
)

func TestFetchTimeout(t *testing.T) {
	// -- create a new logger --
	lg := logger.NewLogger()

	// -- connect to redis --
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// -- create the api handler --
	handler := &Handler{
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

	limit := 202

	wg := &sync.WaitGroup{}

	wg.Add(limit)

	for i := 0; i < limit; i++ {
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			code, _, body, errs := handler.FetchProfile("9032ea59caa14489a167c19a32f9771d")

			if code != fiber.StatusOK {
				t.Error(errs)
				t.Error(code)
				t.Error(body)
			}

			if len(errs) > 0 {
				t.Error(errs)
				t.Error(code)
				t.Error(body)
			}
		}(wg)
	}

	wg.Wait()
}

func TestFetchIP(t *testing.T) {
	// -- create a new logger --
	lg := logger.NewLogger()

	// -- connect to redis --
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// -- create the api handler --
	handler := &Handler{
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

	limit := 8

	wg := &sync.WaitGroup{}

	wg.Add(limit)

	for i := 0; i < limit; i++ {
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			code, body, errs := handler.fetchIP()

			println(string(body))

			if code != fiber.StatusOK {
				t.Error(errs)
				t.Error(code)
				t.Error(body)
			}

			if len(errs) > 0 {
				t.Error(errs)
				t.Error(code)
				t.Error(body)
			}
		}(wg)
	}

	wg.Wait()
}
