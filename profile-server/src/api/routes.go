package api

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/gofiber/fiber/v2"
	"time"
)

const TTL = 15 * time.Minute

func (h *Handler) GetProfile(c *fiber.Ctx) error {
	playerUUID := c.Params("uuid")
	remoteAddr := c.Context().Conn().RemoteAddr().String()

	h.Logger.Info("%s GET /profile/%s", remoteAddr, playerUUID)

	if isValidUUID(playerUUID) {
		//check if the profile exists in redis already and is not expired
		exists, item, err := h.cacheGet(playerUUID)

		//check if a redis error occurred
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		//check if the cache was a hit or miss
		if exists {
			//cache hit
			h.Logger.Info("[%s] Cache Hit for [%s]", playerUUID, remoteAddr)
			c.Status(fiber.StatusOK)
			return c.SendString(item)
		} else {
			//cache miss
			h.Logger.Info("[%s] Cache Miss for %s", playerUUID, remoteAddr)
			code, profileResponse, profileResponseString, errs := h.FetchProfile(playerUUID)

			//check if fetching the profile yielded any errors
			if len(errs) > 0 {
				//log all the errors that occurred
				for _, err := range errs {
					if err != nil {
						h.Logger.Error("%v", err)
					}
				}

				//return the error's status code
				return c.SendStatus(code)
			}

			//check if the profile was able to be fetched
			if profileResponse == nil {
				//log all the errors that occurred
				for _, err := range errs {
					if err != nil {
						h.Logger.Error("%v", err)
					}
				}

				//return the error's status code
				return c.SendStatus(code)
			}

			//cache the profile
			err = h.cachePut(playerUUID, string(profileResponseString))
			if err != nil {
				h.Logger.Error("[%s] Failed to cache profile: %v", playerUUID, err)
			}

			c.Status(code)
			return c.Send(profileResponseString)
		}
	} else {
		c.Status(fiber.StatusBadRequest)
		return c.SendString(fmt.Sprintf("bad uuid: %s", playerUUID))
	}
}

func (h *Handler) GetProfiles(c *fiber.Ctx) error {
	remoteAddr := c.Context().Conn().RemoteAddr().String()
	uuidsBody := new(UUIDSBody)

	//parse the uuids array
	if err := c.BodyParser(uuidsBody); err != nil {
		h.Logger.Error("Failed to parse body: %v from %s", uuidsBody, remoteAddr)
		c.Status(fiber.StatusInternalServerError)
		return err
	}

	uuids := uuidsBody.UUIDS

	//check if all uuids are valid
	for _, playerUUID := range uuids {
		if !isValidUUID(playerUUID) {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("bad uuid: %s", playerUUID))
		}
	}

	//output array
	var profileBodyArray []*ProfileResponse
	var nonCachedUUIDS []string

	//check for cached uuids
	for _, playerUUID := range uuids {
		exists, item, err := h.cacheGet(playerUUID)

		//check if a redis error occured
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if exists {
			//cache hit
			h.Logger.Info("[%s] Cache Hit for [%s]", playerUUID, remoteAddr)

			//add the output to the array
			decodedProfileResponse := &ProfileResponse{}
			err := json.Unmarshal([]byte(item), decodedProfileResponse)
			if err != nil {
				h.Logger.Error("Failed to unmarshall profile!")
				return c.SendStatus(fiber.StatusInternalServerError)
			}

			//aggregate cached ProfileResponses
			profileBodyArray = append(profileBodyArray, decodedProfileResponse)
		} else {
			//cache miss
			h.Logger.Info("[%s] Cache Miss for %s", playerUUID, remoteAddr)

			//add non-cached uuids to be looked up
			nonCachedUUIDS = append(nonCachedUUIDS, playerUUID)
		}
	}

	if len(nonCachedUUIDS) > 0 {
		//lookup non-cached uuids
		responses := h.FetchProfiles(nonCachedUUIDS)

		//check for errors in multi-fetch
		for _, response := range responses {
			if len(response.Errs) > 0 {
				for _, err := range response.Errs {
					h.Logger.Error("%v", err)
				}

				return c.SendStatus(fiber.StatusInternalServerError)
			}

			if response.Profile == nil {
				for _, err := range response.Errs {
					h.Logger.Error("%v", err)
				}

				return c.SendStatus(fiber.StatusInternalServerError)
			}

			if response.Code != fiber.StatusOK {
				for _, err := range response.Errs {
					h.Logger.Error("%v", err)
				}

				return c.SendStatus(response.Code)
			}
		}

		//aggregate new responses and cache them
		for _, response := range responses {
			profileBodyArray = append(profileBodyArray, response.Profile)

			err := h.cachePut(response.Id, string(response.Body))
			if err != nil {
				h.Logger.Error("[%s] Failed to cache profile: %v", response.Id, err)
			}
		}
	}

	out, err := json.Marshal(profileBodyArray)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		h.Logger.Error("%v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "json Marhsal for mojang response failed")
	}

	return c.Send(out)
}

func (h *Handler) GetTexture(c *fiber.Ctx) error {
	textureid := c.Params("textureid")
	remoteAddr := c.Context().Conn().RemoteAddr().String()

	if isValidTextureId(textureid) {
		//check if the texture exists in redis already and is not expired
		item, err := h.Rdb.Get(h.Ctx, textureid).Result()

		switch err {
		case redis.Nil:
			//profile does not exist, fetch the profile
			h.Logger.Info("[%s] Cache Miss for %s", textureid, remoteAddr)
			code, textureBase64, _, errs := h.FetchTexture(textureid)

			//check if fetching the profile yielded any errors
			if len(errs) > 0 {
				//log all the errors that occurred
				for _, err := range errs {
					if err != nil {
						h.Logger.Error("%v", err)
					}
				}

				//return the error's status code
				return c.SendStatus(code)
			}

			//check if the profile was able to be fetched
			if textureBase64 == "" {
				//log all the errors that occurred
				for _, err := range errs {
					if err != nil {
						h.Logger.Error("%v", err)
					}
				}

				//return the error's status code
				return c.SendStatus(code)
			}

			//cache the profile
			err = h.Rdb.Set(h.Ctx, textureid, textureBase64, TTL).Err()

			if err != nil {
				h.Logger.Error("[%s] Failed to cache profile: %v", textureid, err)
			}

			c.Status(code)
			return c.SendString(textureBase64)

		case nil:
			h.Logger.Info("[%s] Cache Hit for [%s]", textureid, remoteAddr)
			c.Status(fiber.StatusOK)
			return c.SendString(item)

		default:
			//some redis error occurred during cache lookup
			h.Logger.Error("%v", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
	} else {
		c.Status(fiber.StatusBadRequest)
		return c.SendString(fmt.Sprintf("bad skinid: %s", textureid))
	}
}

func (h *Handler) GetTextures(c *fiber.Ctx) error {
	remoteAddr := c.Context().Conn().RemoteAddr().String()
	texturesBody := new(TexturesBody)

	//parse the textureids array
	if err := c.BodyParser(texturesBody); err != nil {
		h.Logger.Error("Failed to parse body: %v from %s", texturesBody, remoteAddr)
		c.Status(fiber.StatusInternalServerError)
		return err
	}

	textureids := texturesBody.Textures

	//check if all textureids are valid
	for _, textureid := range textureids {
		if !isValidTextureId(textureid) {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("bad textureid: %s", textureid))
		}
	}

	//output array
	var base64TextureArray []string
	var nonCachedTextureIds []string

	//check for cached textureids
	for _, textureid := range textureids {
		exists, item, err := h.cacheGet(textureid)

		//check if a redis error occured
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if exists {
			//cache hit
			h.Logger.Info("[%s] Cache Hit for [%s]", textureid, remoteAddr)

			//aggregate cached base64textures
			base64TextureArray = append(base64TextureArray, item)
		} else {
			//cache miss
			h.Logger.Info("[%s] Cache Miss for %s", textureid, remoteAddr)

			//add non-cached uuids to be looked up
			nonCachedTextureIds = append(nonCachedTextureIds, textureid)
		}
	}

	if len(nonCachedTextureIds) > 0 {
		//lookup non-cached uuids
		responses := h.FetchTextures(nonCachedTextureIds)

		//check for errors in multi-fetch
		for _, response := range responses {
			if len(response.Errs) > 0 {
				for _, err := range response.Errs {
					h.Logger.Error("%v", err)
				}

				return c.SendStatus(fiber.StatusInternalServerError)
			}

			if response.Base64Texture == "" {
				for _, err := range response.Errs {
					h.Logger.Error("%v", err)
				}

				return c.SendStatus(fiber.StatusInternalServerError)
			}

			if response.Code != fiber.StatusOK {
				for _, err := range response.Errs {
					h.Logger.Error("%v", err)
				}

				return c.SendStatus(response.Code)
			}
		}

		//aggregate new responses and cache them
		for _, response := range responses {
			base64TextureArray = append(base64TextureArray, response.Base64Texture)

			err := h.cachePut(response.Id, response.Base64Texture)
			if err != nil {
				h.Logger.Error("[%s] Failed to cache profile: %v", response.Id, err)
			}
		}
	}

	out, err := json.Marshal(base64TextureArray)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		h.Logger.Error("%v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "json Marhsal for mojang response failed")
	}

	return c.Send(out)
}
