package myerrors

import "errors"

var ErrDBConnClosed = errors.New("failed to connect to db")
