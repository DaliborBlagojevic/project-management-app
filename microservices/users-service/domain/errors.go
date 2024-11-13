package domain

import "errors"

var (
	errConnectionNotFound      error = errors.New("connection not found")
	errConnectionAlreadyExists error = errors.New("user already exists")
	errForeignConnection       error = errors.New("not a user's connection")
	errPostNotFound            error = errors.New("Post not found")
	errUserNotFound            error = errors.New("user not found")
	errUserNotActive           error = errors.New("user not active")
	errInvalidCredentials      error = errors.New("incorrect username or password")
	errInvalidToken            error = errors.New("token invalid")
	errUnauthorized            error = errors.New("unauthorized")
	errUserAlreadyExists		error = errors.New("User with the given username already exists")
	errCodeExpired				error = errors.New("Your activation code has expired or is invalid")
)

func ErrConnectionNotFound() error {
	return errConnectionNotFound
}

func ErrConnectionAlreadyExists() error {
	return errConnectionAlreadyExists
}

func ErrForeignConnection() error {
	return errForeignConnection
}

func ErrPostNotFound() error {
	return errPostNotFound
}

func ErrUserNotFound() error {
	return errUserNotFound
}

func ErrUserNotActive() error {
	return errUserNotActive
}

func ErrInvalidCredentials() error {
	return errInvalidCredentials
}

func ErrInvalidToken() error {
	return errInvalidToken
}

func ErrUnauthorized() error {
	return errUnauthorized
}


func ErrUserAlreadyExists() error {
	return errUserAlreadyExists
}

func ErrCodeExpired() error {
	return errCodeExpired
}
