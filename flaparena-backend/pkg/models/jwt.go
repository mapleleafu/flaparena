package models

import (
	"github.com/golang-jwt/jwt/v4"
)

type CustomClaims struct {
    jwt.RegisteredClaims
    ID       string   `json:"id"`
	Username string `json:"username"`
}
