package middleware

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserData struct {
	Uid   uuid.UUID `json:"uid"`
	Email string    `json:"email"`
	// FirstName  string    `json:"first_name"`
	// SecondName string    `json:"second_name"`
	Role string `json:"role"`
}
type UserClaims struct {
	jwt.RegisteredClaims
	Data UserData `json:"data"`
}
