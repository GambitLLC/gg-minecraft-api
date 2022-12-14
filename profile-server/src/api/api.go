package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/meilisearch/meilisearch-go"
	"net"
	"regexp"
	"sync"
	"sync/atomic"

	"bed.gg/minecraft-api/v2/src/logger"
	"github.com/go-redis/redis/v9"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

type Handler struct {
	Logger   *logger.ZapLogger
	Rdb      *redis.Client
	MSClient *meilisearch.Client
	Ctx      context.Context
	IPPool   []net.IP
	IpIdx    uint32
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

type MultiProfileResponse struct {
	Code    int
	Profile *ProfileResponse
	Body    []byte
	Errs    []error
	Id      string
}

type MultiTextureResponse struct {
	Code          int
	Base64Texture string
	Body          []byte
	Errs          []error
	Id            string
}

type TexturesBody struct {
	Textures []string `json:"textures"`
}

type UUIDSBody struct {
	UUIDS []string `json:"uuids"`
}

// IsValidUUID helper method to check if the provided uuid is a valid minecraft uuid
func IsValidUUID(u string) bool {
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

// isValidTextureId helper method to check if the provided textureid is in a valid format
func isValidTextureId(id string) bool {
	matched, err := regexp.Match("^[a-fA-F0-9]+$", []byte(id))

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

		if h.Logger != nil {
			h.Logger.Error("%v", err)
		}

		return fiber.StatusInternalServerError, nil, []error{err}
	}

	if len(h.IPPool) > 0 {
		customDialer := fasthttp.TCPDialer{
			Concurrency: 1000,
			LocalAddr: &net.TCPAddr{
				IP: h.IPPool[atomic.AddUint32(&h.IpIdx, 1)%uint32(len(h.IPPool))],
			},
		}

		if h.Logger != nil {
			h.Logger.Info("Dialing %s from %s", fmt.Sprintf(formatUrl, args...), customDialer.LocalAddr.String())
		}

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

// FetchProfiles fetches multiple profile jsons concurrently from the mojang api and returns an array of MultiProfileResponse
func (h *Handler) FetchProfiles(uuids []string) []*MultiProfileResponse {
	responses := make(chan *MultiProfileResponse, len(uuids))

	wg := &sync.WaitGroup{}
	wg.Add(len(uuids))

	for _, playerUUID := range uuids {
		go func(playerUUID string, h *Handler, wg *sync.WaitGroup, responses chan *MultiProfileResponse) {
			defer wg.Done()

			code, profileResponse, body, errs := h.FetchProfile(playerUUID)

			response := &MultiProfileResponse{
				Code:    code,
				Profile: profileResponse,
				Body:    body,
				Errs:    errs,
				Id:      playerUUID,
			}

			responses <- response

		}(playerUUID, h, wg, responses)
	}

	wg.Wait()
	close(responses)

	var arr []*MultiProfileResponse
	for response := range responses {
		arr = append(arr, response)
	}

	return arr
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

// FetchTexture fetches the texture as a base64 string from mojang api
func (h *Handler) FetchTexture(textureid string) (int, string, []byte, []error) {
	code, body, errs := h.fetchMojang("https://textures.minecraft.net/texture/%s", textureid)

	if len(errs) > 0 {
		return code, "", nil, errs
	}

	if code != fiber.StatusOK {
		return code, "", nil, errs
	}

	//encode the texture to base64 and return the encoded string
	return code, base64.StdEncoding.EncodeToString(body), body, []error{}
}

// FetchTextures fetches multiple textures concurrently from the mojang api and returns an array of MultiTextureResponse
func (h *Handler) FetchTextures(textureids []string) []*MultiTextureResponse {
	responses := make(chan *MultiTextureResponse, len(textureids))

	wg := &sync.WaitGroup{}
	wg.Add(len(textureids))

	for _, textureid := range textureids {
		go func(textureid string, h *Handler, wg *sync.WaitGroup, responses chan *MultiTextureResponse) {
			defer wg.Done()

			code, base64Texture, body, errs := h.FetchTexture(textureid)

			response := &MultiTextureResponse{
				Code:          code,
				Base64Texture: base64Texture,
				Body:          body,
				Errs:          errs,
				Id:            textureid,
			}

			responses <- response

		}(textureid, h, wg, responses)
	}

	wg.Wait()
	close(responses)

	var arr []*MultiTextureResponse
	for response := range responses {
		arr = append(arr, response)
	}

	return arr
}
