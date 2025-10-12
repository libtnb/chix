package binder

import (
	"errors"
)

// Binder errors
var (
	ErrSuitableContentNotFound = errors.New("binder: suitable content not found to parse body")
	ErrMapNotConvertible       = errors.New("binder: map is not convertible to map[string]string or map[string][]string")
)

// Init default binders for Fiber
var (
	HeaderBinder = &headerBinding{}
	CookieBinder = &cookieBinding{}
	QueryBinder  = &queryBinding{}
	FormBinder   = &formBinding{}
	URIBinder    = &uriBinding{}
	XMLBinder    = &xmlBinding{}
	JSONBinder   = &jsonBinding{}
)
