package binder

import (
	"errors"
)

// Binding source tags, also used as the struct tag alias for each source.
const (
	bindingURI    = "uri"
	bindingForm   = "form"
	bindingQuery  = "query"
	bindingHeader = "header"
	bindingCookie = "cookie"
)

// Binder errors
var (
	ErrSuitableContentNotFound = errors.New("binder: suitable content not found to parse body")
	ErrMapNotConvertible       = errors.New("binder: map is not convertible to map[string]string or map[string][]string")
	ErrMapNilDestination       = errors.New("binder: map destination is nil and cannot be initialized")
	ErrInvalidDestinationValue = errors.New("binder: invalid destination value")
	ErrUnmatchedBrackets       = errors.New("unmatched brackets")
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
