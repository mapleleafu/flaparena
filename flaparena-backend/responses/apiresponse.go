package responses

// APIError interface for custom API errors
type APIError interface {
    Error() string
    StatusCode() int
}

type BadRequestError struct {
    Msg string
}

func (e BadRequestError) Error() string {
    return e.Msg
}

func (BadRequestError) StatusCode() int {
    return 400
}

type UnauthorizedError struct {
	Msg string
}

func (e UnauthorizedError) Error() string {
	return e.Msg
}

func (UnauthorizedError) StatusCode() int {
	return 401
}

type NotFoundError struct {
	Msg string
}

func (e NotFoundError) Error() string {
	return e.Msg
}

func (NotFoundError) StatusCode() int {
	return 404
}

type InternalServerError struct {
	Msg string
}

func (e InternalServerError) Error() string {
	return e.Msg
}

func (InternalServerError) StatusCode() int {
	return 500
}

