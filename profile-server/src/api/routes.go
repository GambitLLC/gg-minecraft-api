package api

import (
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
		item, err := h.Rdb.Get(h.Ctx, playerUUID).Result()

		switch err {
		case redis.Nil:
			//profile does not exist, fetch the profile
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
			err = h.Rdb.Set(h.Ctx, playerUUID, profileResponseString, TTL).Err()

			if err != nil {
				h.Logger.Error("[%s] Failed to cache profile: %v", playerUUID, err)
			}

			c.Status(code)
			return c.Send(profileResponseString)

		case nil:
			h.Logger.Info("[%s] Cache Hit for [%s]", playerUUID, remoteAddr)
			c.Status(fiber.StatusOK)
			return c.SendString(item)

		default:
			//some redis error occurred during cache lookup
			h.Logger.Error("%v", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
	} else {
		c.Status(fiber.StatusBadRequest)
		return c.SendString(fmt.Sprintf("bad uuid: %s", playerUUID))
	}
}

//func (h *Handler) GetUUID(c *fiber.Ctx) error {
//	username := c.Params("username")
//
//	if isValidUsername(username) {
//		code, _, uuidResponseString, errs := h.FetchUUID(username)
//
//		if len(errs) > 0 {
//			for _, err := range errs {
//				if err != nil {
//					h.Logger.Error("%v", err)
//				}
//			}
//			return c.SendStatus(fiber.StatusInternalServerError)
//		}
//
//		c.Status(code)
//		return c.Send(uuidResponseString)
//	} else {
//		c.Status(fiber.StatusBadRequest)
//		return c.SendString(fmt.Sprintf("bad username: %s", username))
//	}
//}

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
