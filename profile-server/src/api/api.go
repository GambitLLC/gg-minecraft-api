package api

import (
	"bed.gg/minecraft-api/v2/src/logger"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"net"
	"regexp"
	"sync/atomic"
)

type Handler struct {
	Logger *logger.ZapLogger
	Rdb    *redis.Client
	Ctx    context.Context
	IPPool []net.IP
	IpIdx  uint32
}

type ProfileResponse struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Properties []struct {
		Name      string `json:"name"`
		Value     string `json:"value"`
		Signature string `json:"signature"`
	} `json:"properties"`
}

type UsernameResponse struct {
	Name         string `json:"name"`
	Id           string `json:"id"`
	Error        string `json:"error"`
	ErrorMessage string `json:"errorMessage"`
}

// isValidUUID helper method to check if the provided uuid is a valid minecraft uuid
func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

// isValidUsername helper method to check if the provided username is a valid minecraft username
func isValidUsername(u string) bool {
	matched, err := regexp.Match("^[a-zA-Z0-9_]{2,16}$", []byte(u))

	if err != nil {
		return false
	}

	return matched
}

func (h *Handler) fetchIP() (int, []byte, []error) {
	a := fiber.AcquireAgent()
	req := a.Request()
	req.Header.SetMethod(fiber.MethodGet)
	req.SetRequestURI(fmt.Sprintf("https://ipinfo.io/json"))

	if err := a.Parse(); err != nil {
		h.Logger.Error("%v", err)
		return fiber.StatusInternalServerError, nil, []error{err}
	}

	if len(h.IPPool) > 0 {
		customDialer := fasthttp.TCPDialer{
			Concurrency: 1000,
			LocalAddr: &net.TCPAddr{
				IP: h.IPPool[atomic.AddUint32(&h.IpIdx, 1)%uint32(len(h.IPPool))],
			},
		}

		h.Logger.Info("Dialing %s from %s", fmt.Sprintf("https://ipinfo.io/json"), customDialer.LocalAddr.String())

		a.HostClient.Dial = func(addr string) (net.Conn, error) {
			return customDialer.Dial(addr)
		}
	}

	return a.Bytes()
}

// fetchMojang helper method to access mojang api
func (h *Handler) fetchMojang(formatUrl string, args ...interface{}) (int, []byte, []error) {
	a := fiber.AcquireAgent()
	req := a.Request()
	req.Header.SetMethod(fiber.MethodGet)
	req.SetRequestURI(fmt.Sprintf(formatUrl, args...))

	if err := a.Parse(); err != nil {
		h.Logger.Error("%v", err)
		return fiber.StatusInternalServerError, nil, []error{err}
	}

	if len(h.IPPool) > 0 {
		customDialer := fasthttp.TCPDialer{
			Concurrency: 1000,
			LocalAddr: &net.TCPAddr{
				IP: h.IPPool[atomic.AddUint32(&h.IpIdx, 1)%uint32(len(h.IPPool))],
			},
		}

		h.Logger.Info("Dialing %s from %s", fmt.Sprintf(formatUrl, args...), customDialer.LocalAddr.String())

		a.HostClient.Dial = func(addr string) (net.Conn, error) {
			return customDialer.Dial(addr)
		}
	}

	return a.Bytes()
}

// FetchProfile fetches the profile json from mojang api and returns a ProfileResponse
func (h *Handler) FetchProfile(playerUUID string) (int, *ProfileResponse, []byte, []error) {
	code, body, errs := h.fetchMojang("https://sessionserver.mojang.com/session/minecraft/profile/%s?unsigned=false", playerUUID)

	if len(errs) > 0 {
		return code, nil, nil, errs
	}

	if code != fiber.StatusOK {
		return code, nil, nil, errs
	}

	//deserialize the body and return the ProfileResponse
	profileResponse := &ProfileResponse{}
	err := json.Unmarshal(body, profileResponse)

	if err != nil {
		return code, nil, nil, []error{err}
	}

	return code, profileResponse, body, []error{}
}

// FetchUUID fetches the username json from mojang api and returns a UsernameResponse
func (h *Handler) FetchUUID(username string) (int, *UsernameResponse, []byte, []error) {
	code, body, errs := h.fetchMojang("https://api.mojang.com/users/profiles/minecraft/%s", username)

	if len(errs) > 0 {
		return code, nil, nil, errs
	}

	if code != fiber.StatusOK {
		return code, nil, nil, errs
	}

	//deserialize the body and return the UsernameResponse
	usernameResponse := &UsernameResponse{}
	err := json.Unmarshal(body, usernameResponse)

	if err != nil {
		return code, nil, nil, []error{err}
	}

	return code, usernameResponse, body, []error{}
}
