package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/doingodswork/deflix-stremio/pkg/debrid/alldebrid"
	"github.com/doingodswork/deflix-stremio/pkg/debrid/premiumize"
	"github.com/doingodswork/deflix-stremio/pkg/debrid/realdebrid"
)

// createAuthMiddleware creates a middleware that checks the validity of RealDebrid, AllDebrid and Premiumize API tokens/keys as well as Premiumize OAuth2 data.
func createAuthMiddleware(rdClient *realdebrid.Client, adClient *alldebrid.Client, pmClient *premiumize.Client, useOAUTH2 bool, confRD, confPM oauth2.Config, aesKey []byte, logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rCtx := c.Context()
		udString := c.Params("userData", "")
		if udString == "" {
			// Should never occur, because the manifest states that configuration is required and go-stremio's route matcher middleware filters these out.
			logger.Error("User data is empty, but this should have been handled by go-stremio's router matcher middleware alraedy")
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		userData, err := decodeUserData(udString, logger)
		if err != nil {
			// The error is already logged in the decodeUserData function.
			// It's most likely a client-side encoding error.
			return c.SendStatus(fiber.StatusBadRequest)
		}

		if useOAUTH2 {
			if userData.RDoauth2 == "" && userData.ADkey == "" && userData.PMoauth2 == "" {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
			// We expect a user to have *either* RD OAuth2 data *or* an AD key *or* Premiumize OAuth2 data
			if userData.RDoauth2 != "" {
				accessToken, err := getAccessTokenForOAuth2data(c, confRD, aesKey, userData.RDoauth2, logger)
				if err != nil {
					// HTTP responses are already handled
					return err
				}
				if err = rdClient.TestToken(c.Context(), accessToken); err != nil {
					logger.Info("Access token is invalid or validation failed", zap.Error(err))
					return c.SendStatus(fiber.StatusForbidden)
				}
				c.Locals("deflix_keyOrToken", accessToken)
			} else if userData.ADkey != "" {
				if err := adClient.TestAPIkey(rCtx, userData.ADkey); err != nil {
					logger.Info("API key is invalid or validation failed", zap.Error(err))
					return c.SendStatus(fiber.StatusForbidden)
				}
				c.Locals("deflix_keyOrToken", userData.ADkey)
			} else if userData.PMoauth2 != "" {
				accessToken, err := getAccessTokenForOAuth2data(c, confPM, aesKey, userData.PMoauth2, logger)
				if err != nil {
					// HTTP responses are already handled
					return err
				}
				if err = pmClient.TestAPIkey(c.Context(), accessToken); err != nil {
					logger.Info("Access token is invalid or validation failed", zap.Error(err))
					return c.SendStatus(fiber.StatusForbidden)
				}
				c.Locals("deflix_keyOrToken", accessToken)
			}
		} else {
			if userData.RDtoken == "" && userData.ADkey == "" && userData.PMkey == "" {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
			// We expect a user to have *either* an RD token *or* an AD key *or* a Premiumize key
			if userData.RDtoken != "" {
				if err := rdClient.TestToken(rCtx, userData.RDtoken); err != nil {
					logger.Info("API key is invalid or validation failed", zap.Error(err))
					return c.SendStatus(fiber.StatusForbidden)
				}
				c.Locals("deflix_keyOrToken", userData.RDtoken)
			} else if userData.ADkey != "" {
				if err := adClient.TestAPIkey(rCtx, userData.ADkey); err != nil {
					logger.Info("API key is invalid or validation failed", zap.Error(err))
					return c.SendStatus(fiber.StatusForbidden)
				}
				c.Locals("deflix_keyOrToken", userData.ADkey)
			} else if userData.PMkey != "" {
				if err := pmClient.TestAPIkey(rCtx, userData.PMkey); err != nil {
					logger.Info("API key is invalid or validation failed", zap.Error(err))
					return c.SendStatus(fiber.StatusForbidden)
				}
				c.Locals("deflix_keyOrToken", userData.PMkey)
			}
		}

		return c.Next()
	}
}

func getAccessTokenForOAuth2data(c *fiber.Ctx, conf oauth2.Config, aesKey []byte, oauth2data string, logger *zap.Logger) (string, error) {
	ciphertext, err := base64.RawURLEncoding.DecodeString(oauth2data)
	if err != nil {
		// It's most likely a client-side encoding error
		return "", c.SendStatus(fiber.StatusBadRequest)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		logger.Warn("Couldn't create block cipher from AES key", zap.Error(err))
		return "", c.SendStatus(fiber.StatusInternalServerError)
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		logger.Error("Couldn't create AES GCM", zap.Error(err))
		return "", c.SendStatus(fiber.StatusInternalServerError)
	}
	// The nonce is prepended
	nonce := ciphertext[:aesgcm.NonceSize()]
	ciphertext = ciphertext[aesgcm.NonceSize():]

	tokenJSON, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", c.SendStatus(fiber.StatusForbidden)
	}
	token := &oauth2.Token{}
	if err = json.Unmarshal(tokenJSON, token); err != nil {
		// How likely is it that if the previous decoding worked, that it's now the client's fault vs ours?
		return "", c.SendStatus(fiber.StatusBadRequest)
	}
	tokenSource := conf.TokenSource(c.Context(), token)
	// The token source automatically refreshes the token with the refresh token
	validToken, err := tokenSource.Token()
	if err != nil {
		return "", c.SendStatus(fiber.StatusForbidden)
	}

	return validToken.AccessToken, nil
}
