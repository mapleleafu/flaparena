package models

type ApiResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data"`
    Error   interface{} `json:"error"`
}

func SuccessResponse(data interface{}) ApiResponse {
    return ApiResponse{Success: true, Data: data, Error: nil}
}

func ErrorResponse(errorMessage string) ApiResponse {
    return ApiResponse{Success: false, Data: nil, Error: errorMessage}
}
