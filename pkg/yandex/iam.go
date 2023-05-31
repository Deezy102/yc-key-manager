package yandex

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

type IAM struct {
	token string
}

type Config struct {
	ServiceAccountID string
	KeyFile          string
	KeyID            string
}

func NewIAM(config *Config) (*IAM, error) {
	iam, err := getIAMToken(config)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create iam token")
	}

	i := &IAM{token: iam}

	return i, nil
}

func (i *IAM) Value() string {
	return i.token
}

func loadPrivateKey(keyFile string) (*rsa.PrivateKey, error) {
	file, err := os.Open(keyFile)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	rsaPrivateKey, err := jwt.ParseRSAPrivateKeyFromPEM(data)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return rsaPrivateKey, nil
}

func signedToken(conf *Config) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    conf.ServiceAccountID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		Audience:  []string{"https://iam.api.cloud.yandex.net/iam/v1/tokens"},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodPS256, claims)
	token.Header["kid"] = conf.KeyID
	privateKey, err := loadPrivateKey(conf.KeyFile)
	if err != nil {
		return "", errors.WithStack(err)
	}
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", errors.New(err.Error())
	}
	return signed, nil
}

func getIAMToken(conf *Config) (string, error) {
	jwt, err := signedToken(conf)
	if err != nil {
		return "", errors.Wrap(err, "unable to sign token")
	}
	resp, err := http.Post(
		"https://iam.api.cloud.yandex.net/iam/v1/tokens",
		"application/json",
		strings.NewReader(fmt.Sprintf(`{"jwt":"%s"}`, jwt)),
	)
	if err != nil {
		return "", errors.New(err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", errors.Errorf("%s: %s", resp.Status, body)
	}
	var data struct {
		IAMToken string `json:"iamToken"`
	}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", errors.New("decoding error")
	}
	return data.IAMToken, nil
}
