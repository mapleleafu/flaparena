package middleware

import (
    "context"
    "net/http"
    "strings"

    "github.com/golang-jwt/jwt/v4"
    "github.com/mapleleafu/flaparena/flaparena-backend/models"
    "github.com/mapleleafu/flaparena/flaparena-backend/utils"
    "github.com/mapleleafu/flaparena/flaparena-backend/responses"
)

func JWTValidationMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tokenStr := r.Header.Get("Authorization")
        tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

        keyFunc := func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, jwt.ErrInvalidKey
            }
            return []byte("secret"), nil
        }

        token, err := jwt.ParseWithClaims(tokenStr, &models.CustomClaims{}, keyFunc)
        if err != nil || !token.Valid {
            utils.HandleError(w, responses.UnauthorizedError{Msg: "You are not authorized to access this resource."})
            return
        }

        authInfo, ok := token.Claims.(*models.CustomClaims)
        if !ok {
            utils.HandleError(w, responses.InternalServerError{Msg: "Error processing request."})
            return
        }

        // Store the claims in the context
        ctx := context.WithValue(r.Context(), "authInfo", authInfo)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
