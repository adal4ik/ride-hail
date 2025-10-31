package myerrors

import "errors"

var (
	ErrDBConnClosed    = errors.New("failed to connect to db")
	ErrDBConnClosedMsg = errors.New("internal error, please try again later")
)
