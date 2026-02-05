package jwt

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	// "github.com/google/uuid"
	// "github.com/krassor/skygrow/backend-service-calendar/internal/models/domain"
)

var (
	ErrorClaimsTypeAssertion = errors.New("error claims type assertion")
	ErrorNoExpiresAtClaim    = errors.New("error exp claim not found in the jwt token")
)

// func ParseToken(tokenString string, secret string) (domain.CalendarUser, error) {
// 	op := "utils.jwt.ParseToken()"
// 	log := slog.With(
// 		slog.String("op", op),
// 	)

// 	var validMethods = []string{jwt.SigningMethodHS256.Name}

// 	//token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
// 	//	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
// 	//		return nil, fmt.Errorf("%s:%w", op, errors.New("unexpected signing method"))
// 	//	}
// 	//	return []byte(secret), nil
// 	//}, jwt.WithValidMethods(validMethods))

// 	token, err := jwt.Parse(
// 		tokenString,
// 		func(token *jwt.Token) (interface{}, error) {
// 			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
// 				return nil, fmt.Errorf("%s:%w", op, errors.New("unexpected signing method"))
// 			}
// 			return []byte(secret), nil
// 		},
// 		jwt.WithValidMethods(validMethods),
// 	)

// 	if err != nil {
// 		return domain.CalendarUser{}, fmt.Errorf("%s:%w", op, err)
// 	}

// 	claims, ok := token.Claims.(jwt.MapClaims)
// 	if !ok {
// 		return domain.CalendarUser{}, fmt.Errorf("%s:%w", op, ErrorClaimsTypeAssertion)
// 	}

// 	// validate the essential claims
// 	if !token.Valid {
// 		return domain.CalendarUser{}, fmt.Errorf("%s:%w", op, errors.New("invalid token"))
// 	}

// 	userDataMap := claims["data"].(map[string]interface{})
// 	uid, err := uuid.Parse(userDataMap["uid"].(string))
// 	if err != nil {
// 		return domain.CalendarUser{}, fmt.Errorf("%s:%w", op, err)
// 	}

// 	userData := UserData{
// 		Uid:        uid,
// 		Email:      userDataMap["email"].(string),
// 		FirstName:  userDataMap["first_name"].(string),
// 		SecondName: userDataMap["second_name"].(string),
// 		Role:       userDataMap["role"].(string),
// 	}

// 	et, err := claims.GetExpirationTime()
// 	if err != nil {
// 		return domain.CalendarUser{}, fmt.Errorf("%s:%w", op, err)
// 	}
// 	if et == nil {
// 		return domain.CalendarUser{}, fmt.Errorf("%s:%w", op, ErrorNoExpiresAtClaim)
// 	}

// 	log.Debug(
// 		"",
// 		slog.String("UserData", fmt.Sprintf("%+v", userData)),
// 		slog.String("ExpirationTime", et.String()),
// 		slog.String("TimeNow", time.Now().String()),
// 	)

// 	if et.Before(time.Now()) {
// 		return domain.CalendarUser{}, fmt.Errorf("%s:%w", op, errors.New("token expired"))
// 	}

// 	user := domain.CalendarUser{
// 		BaseModel:  domain.BaseModel{ID: userData.Uid},
// 		FirstName:  userData.FirstName,
// 		SecondName: userData.SecondName,
// 		Email:      userData.Email,
// 	}

// 	return user, nil
// }

func ParseAndValidateToken[T any](tokenString string, secret string) (*T, error) {
	op := "utils.jwt.ParseAndValidateToken()"
	log := slog.With(
		slog.String("op", op),
	)

	// Парсим токен с проверкой подписи
	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			// Проверяем алгоритм подписи
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("%s: unexpected signing method: %v", op, token.Header["alg"])
			}
			return []byte(secret), nil
		})

	if err != nil {
		return nil, fmt.Errorf("%s: token parsing failed: %w", op, err)
	}

	// Проверяем валидность токена
	if !token.Valid {
		return nil, fmt.Errorf("%s: invalid token", op)
	}

	// Извлекаем claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("%s: invalid claims format", op)
	}

	// Конвертируем claims в целевую структуру через JSON маршалинг
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("claims marshaling error: %w", err)
	}

	// Проверяем exp claim
	et, err := claims.GetExpirationTime()
	if err != nil {
		return nil, fmt.Errorf("%s:%w", op, err)
	}
	if et == nil {
		return nil, fmt.Errorf("%s:%w", op, ErrorNoExpiresAtClaim)
	}
	log.Debug(
		"",
		slog.String("UserData", fmt.Sprintf("%+v", claimsJSON)),
		slog.String("ExpirationTime", et.String()),
		slog.String("TimeNow", time.Now().String()),
	)

	if et.Before(time.Now()) {
		return nil, fmt.Errorf("%s:%w", op, errors.New("token expired"))
	}

	var result T
	if err := json.Unmarshal(claimsJSON, &result); err != nil {
		return nil, fmt.Errorf("%s: claims mapping error: %w", op, err)
	}

	return &result, nil
}
