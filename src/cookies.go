package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

func setCookie(name, data string, expires time.Time, w http.ResponseWriter) {
	cookie := http.Cookie{
		Name:     name,
		Value:    data,
		HttpOnly: true,
		Expires:  expires,
		Path:     "/",
		SameSite: http.SameSiteLaxMode, // required to allow oauth
	}
	http.SetCookie(w, &cookie)
}

func clearCookie(name string, w http.ResponseWriter) {
	cookie := http.Cookie{
		Name:     name,
		Value:    "",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
}

func setEncryptedCookie(name, unencodedData string, w http.ResponseWriter) (err error) {
	now := time.Now()
	data := now.Format(time.RFC3339) + "|" + unencodedData
	cipher, err := encryptGCM([]byte(data), []byte(config.EncryptionSecret))
	if err != nil {
		return err
	}
	b64 := base64.StdEncoding.EncodeToString(cipher)

	setCookie(name, b64, now.Add(config.CookieAgeDuration()), w)
	return nil
}

func readEncryptedCookie(name string, r *http.Request) (unencodedData string, err error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}

	b64 := cookie.Value

	cipher, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}

	dataB, err := decrypt(cipher, []byte(config.EncryptionSecret))
	if err != nil {
		return "", err
	}
	data := string(dataB)

	split := strings.Index(data, "|")
	if split == -1 {
		return "", errors.New("invalid cookie")
	}

	issued, err := time.Parse(time.RFC3339, data[:split])
	if err != nil {
		return "", errors.New("invalid cookie")
	}

	if time.Now().Sub(issued) > config.CookieAgeDuration() {
		return "", errors.New("expired cookie")
	}

	return data[split+1:], nil
}

func encryptGCM(plaintext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
