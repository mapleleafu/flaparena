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
    "github.com/mapleleafu/flaparena/flaparena-backend/pkg/utils"
    "github.com/mapleleafu/flaparena/flaparena-backend/pkg/responses"
	"github.com/mapleleafu/flaparena/flaparena-backend/pkg/repository"
	"golang.org/x/crypto/bcrypt"
)

func Register(w http.ResponseWriter, r *http.Request) {
    db := repository.ConnectToDB(config.LoadConfig())
    defer db.Close()

    var user models.User
    err := json.NewDecoder(r.Body).Decode(&user)
    if err != nil {
        utils.HandleError(w, responses.BadRequestError{Msg: "Invalid request."})
        return
    }

    if len(user.Username) < 3 || len(user.Username) > 50 {
        utils.HandleError(w, responses.BadRequestError{Msg: "Username must be between 3 and 50 characters."})
        return
    }

    if len(user.Password) < 3 || len(user.Password) > 50 {
        utils.HandleError(w, responses.BadRequestError{Msg: "Password must be between 3 and 50 characters."})
        return
    }

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
    if err != nil {
        utils.HandleError(w, responses.InternalServerError{Msg: "Failed to hash password."})
        return
    }
    user.Password = string(hashedPassword)

    _, err = db.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", user.Username, user.Password)
    if err != nil {
        log.Println(err)
        utils.HandleError(w, responses.InternalServerError{Msg: "Failed to create user."})
        return
    }

    utils.HandleSuccess(w, models.SuccessResponse(map[string]string{"message": "User created successfully."}))
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
            utils.HandleError(w, responses.UnauthorizedError{Msg: "You are not authorized to access this resource."})
            return
        }
        log.Println(err)
        utils.HandleError(w, responses.InternalServerError{Msg: "An error occurred while processing your request."})
        return
    }

    err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginInfo.Password))
    if err != nil {
        utils.HandleError(w, responses.BadRequestError{Msg: "Invalid username or password."})
        return
    }

    // Generate JWT token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "username": user.Username,
        "exp":      time.Now().Add(time.Hour * 72).Unix(),
    })

    tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
    if err != nil {
        utils.HandleError(w, responses.InternalServerError{Msg: "Failed to generate token."})
        return
    }

    refreshToken, err := generateRefreshToken()
    if err != nil {
        utils.HandleError(w, responses.InternalServerError{Msg: "Failed to generate refresh token."})
        return
    }
    
    userID := user.ID
    expiresAt := time.Now().Add(24 * time.Hour * 180) // Expires in 180 days
    
    _, err = db.Exec("INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)",
        userID, refreshToken, expiresAt)
    if err != nil {
        log.Println(err)
        utils.HandleError(w, responses.InternalServerError{Msg: "Failed to store refresh token."})
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
    utils.HandleSuccess(w, models.SuccessResponse(map[string]string{"access_token": tokenString}))
}

func generateRefreshToken() (string, error) {
    tokenBytes := make([]byte, 64) // 64 bytes
    if _, err := rand.Read(tokenBytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

func Logout(w http.ResponseWriter, r *http.Request) {
    refreshTokenCookie, err := r.Cookie("refresh_token")

    db := repository.ConnectToDB(config.LoadConfig())
    defer db.Close()

    if err == nil {
        _, dbErr := db.Exec("DELETE FROM refresh_tokens WHERE token = $1", refreshTokenCookie.Value)
        if dbErr != nil {
            log.Println(dbErr)
            utils.HandleError(w, responses.InternalServerError{Msg: "Failed to delete refresh token."})
        }
    }

    // Expire the cookie to force the client to delete it
    newCookie := &http.Cookie{
        Name:     "refresh_token",
        Value:    "",
        Path:     "/",
        Expires:  time.Now().AddDate(0, 0, -1), 
        MaxAge:   -1,
        HttpOnly: true,
        Secure:   true,
    }
    http.SetCookie(w, newCookie)
    
    utils.HandleSuccess(w, models.SuccessResponse(map[string]string{"message": "Logged out successfully."}))
}

func RefreshToken(w http.ResponseWriter, r *http.Request) {
    refreshTokenCookie, err := r.Cookie("refresh_token")
    if err != nil {
        utils.HandleError(w, responses.UnauthorizedError{Msg: "No refresh token found."})
        return
    }

    db := repository.ConnectToDB(config.LoadConfig())
    defer db.Close()

    var userID int
    var expiresAt time.Time
    err = db.QueryRow("SELECT user_id, expires_at FROM refresh_tokens WHERE token = $1", refreshTokenCookie.Value).Scan(&userID, &expiresAt)
    if err != nil {
        log.Println(err)
        utils.HandleError(w, responses.UnauthorizedError{Msg: "Invalid refresh token."})
        return
    }
    
    if time.Now().After(expiresAt) {
        utils.HandleError(w, responses.UnauthorizedError{Msg: "Refresh token has expired."})
        return
    }

    var user models.User
    err = db.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&user.Username)
    if err != nil {
        log.Println(err)
        utils.HandleError(w, responses.InternalServerError{Msg: "An error occurred while processing your request."})
        return
    }

    // Generate JWT token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "username": user.Username,
        "exp":      time.Now().Add(time.Hour * 72).Unix(),
    })

    tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
    if err != nil {
        utils.HandleError(w, responses.InternalServerError{Msg: "Failed to generate token."})
        return
    }

    utils.HandleSuccess(w, models.SuccessResponse(map[string]string{"access_token": tokenString}))
}
