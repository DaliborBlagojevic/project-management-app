package domain

import "errors"

var (
	errConnectionNotFound      error = errors.New("connection not found")
	errConnectionAlreadyExists error = errors.New("connection already exists")
	errForeignConnection       error = errors.New("not a user's connection")
	errPostNotFound            error = errors.New("post not found")
	errUserNotFound            error = errors.New("user not found")
	errInvalidCredentials      error = errors.New("incorrect username or password")
	errInvalidToken            error = errors.New("token invalid")
	errUnauthorized            error = errors.New("unauthorized")
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

func ErrInvalidCredentials() error {
	return errInvalidCredentials
}

func ErrInvalidToken() error {
	return errInvalidToken
}

func ErrUnauthorized() error {
	return errUnauthorized
}
