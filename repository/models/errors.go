package models

import "net/http"

type ValidationError struct {
	msg string
}

func (e ValidationError) Error() string   { return e.msg }
func (e ValidationError) Message() string { return e.msg }
func (e ValidationError) StatusCode() int { return http.StatusBadRequest }
