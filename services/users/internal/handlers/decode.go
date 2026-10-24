package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/andrew-craig/cairn-reader/pkg/api"
)

// errNULInString is returned by decodeJSONBody when a decoded string contains
// a NUL byte. PostgreSQL text columns cannot store U+0000, so such a value
// would otherwise reach the driver and surface as a 500.
var errNULInString = errors.New("string fields must not contain NUL characters")

// decodeJSONBody decodes r's JSON body (capped at maxRequestBodySize) into dst,
// a pointer to a request struct. On failure it writes the 4xx response and
// returns the error, so the caller only needs to return.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, api.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return err
		}
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
		return err
	}

	if containsNUL(reflect.ValueOf(dst)) {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+errNULInString.Error(), nil, "v1")
		return errNULInString
	}
	return nil
}

// containsNUL reports whether any string reachable from v (through pointers,
// struct fields, slice/array elements, and map keys and values) contains a
// NUL byte.
func containsNUL(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return strings.ContainsRune(v.String(), 0)
	case reflect.Pointer, reflect.Interface:
		return !v.IsNil() && containsNUL(v.Elem())
	case reflect.Struct:
		for i := range v.NumField() {
			if containsNUL(v.Field(i)) {
				return true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if containsNUL(v.Index(i)) {
				return true
			}
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if containsNUL(iter.Key()) || containsNUL(iter.Value()) {
				return true
			}
		}
	}
	return false
}
