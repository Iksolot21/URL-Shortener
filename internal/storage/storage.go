package storage

import "errors"

var (
	ErrURLNotFound = errors.New("url not found")
	ErrURLExists   = errors.New("url exists")
)

type URLSaverURLGetter interface {
	SaveURL(urlToSave string, alias string) error
	GetURL(alias string) (string, error)
}
