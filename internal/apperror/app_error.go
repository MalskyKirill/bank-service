package apperror

type AppError struct {
	StatusCode int
	Message    string
}

func New(statusCode int, message string) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Message:    message,
	}
}

func (e *AppError) Error() string {
	return e.Message
}
