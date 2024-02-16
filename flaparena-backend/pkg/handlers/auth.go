package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
    "crypto/rand"
    "encoding/base64"

	"github.com/golang-jwt/jwt/v4"
	"github.com/mapleleafu/flaparena/flaparena-backend/pkg/config"
	"github.com/mapleleafu/flaparena/flaparena-backend/pkg/models"
	"github.com/mapleleafu/flaparena/flaparena-backend/pkg/repository"
	"golang.org/x/crypto/bcrypt"
)

func Register(w http.ResponseWriter, r *http.Request) {
    db := repository.ConnectToDB(config.LoadConfig())
    defer db.Close()

    var user models.User
    err := json.NewDecoder(r.Body).Decode(&user)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
    if err != nil {
        http.Error(w, "Failed to hash password", http.StatusInternalServerError)
        return
    }
    user.Password = string(hashedPassword)

    _, err = db.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", user.Username, user.Password)
    if err != nil {
        log.Println(err)
        http.Error(w, "Failed to create user", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode("User created successfully")
}

func Login(w http.ResponseWriter, r *http.Request) {
    db := repository.ConnectToDB(config.LoadConfig())
    defer db.Close()

    var loginInfo models.User
    err := json.NewDecoder(r.Body).Decode(&loginInfo)
    if err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    var user models.User
    err = db.QueryRow("SELECT id, username, password FROM users WHERE username = $1", loginInfo.Username).Scan(&user.ID, &user.Username, &user.Password)
    if err != nil {
        if err == sql.ErrNoRows {
            http.Error(w, "Invalid credentials", http.StatusUnauthorized)
            return
        }
        log.Println(err)
        http.Error(w, "Failed to query user", http.StatusInternalServerError)
        return
    }

    err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginInfo.Password))
    if err != nil {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    // Generate JWT token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "username": user.Username,
        "exp":      time.Now().Add(time.Hour * 72).Unix(),
    })

    tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
    if err != nil {
        http.Error(w, "Failed to generate token", http.StatusInternalServerError)
        return
    }

    refreshToken, err := generateRefreshToken()
    if err != nil {
        http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
        return
    }
    
    userID := user.ID
    expiresAt := time.Now().Add(24 * time.Hour * 180) // Expires in 180 days
    
    _, err = db.Exec("INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)",
        userID, refreshToken, expiresAt)
    if err != nil {
        log.Println(err)
        http.Error(w, "Failed to store refresh token", http.StatusInternalServerError)
        return
    }
    
    // Create a cookie
    refreshTokenCookie := &http.Cookie{
        Name:     "refresh_token",
        Value:    refreshToken,
        Path:     "/",
        Expires:  time.Now().Add(24 * time.Hour * 180),
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    }

    // Set the cookie in the response header
    http.SetCookie(w, refreshTokenCookie)

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(tokenString)

}

func generateRefreshToken() (string, error) {
    tokenBytes := make([]byte, 64) // 64 bytes
    if _, err := rand.Read(tokenBytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(tokenBytes), nil
}
