package utils

import (
    "encoding/json"
    "net/http"
    "github.com/mapleleafu/flaparena/flaparena-backend/models"
    "github.com/mapleleafu/flaparena/flaparena-backend/responses"
)

func HandleSuccess(w http.ResponseWriter, response models.ApiResponse) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}

// HandleError checks the error type and sends an appropriate response
func HandleError(w http.ResponseWriter, err error) {
    var statusCode int
    var errorMsg string

    // Type assertion to check if it's a custom API error
    if apiErr, ok := err.(responses.APIError); ok {
        statusCode = apiErr.StatusCode()
        errorMsg = apiErr.Error()
    } else {
        // Default to internal server error if not a custom API error
        statusCode = http.StatusInternalServerError
        errorMsg = "Internal Server Error"
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(models.ApiResponse{Success: false, Data: nil, Error: errorMsg})
}