package utils

import (
	"errors"
	"fmt"
	"time"
)
import "github.com/golang-jwt/jwt/v5"

// todo - вынести в конфиг
var (
	jwtSecret       = []byte("my_super_secret_key")
	tokenExpiration = 24 * time.Hour
)

func GenerateJWT(login string) (string, error) {
	claims := jwt.MapClaims{
		"sub": login,
		"exp": time.Now().Add(tokenExpiration).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ParseJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
