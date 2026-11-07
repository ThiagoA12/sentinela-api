package main

import (
	"encoding/json"
	"errors"
	"io"
	"maps"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/google/uuid"
)

const maxBodyBytes = 1 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return apperror.UnsupportedMedia("Content-Type deve ser application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return decodeError(err)
		}
		return apperror.Invalid("o corpo deve ter um unico objeto json")
	}
	return nil
}

func decodeError(err error) error {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return apperror.TooLarge("corpo maior que 1MB")
	}
	if errors.Is(err, io.EOF) {
		return apperror.Invalid("corpo vazio")
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		if typeErr.Field != "" {
			return apperror.Invalid("campo " + typeErr.Field + " com tipo invalido")
		}
		return apperror.Invalid("o corpo deve ser um objeto JSON")
	}

	if field, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
		return apperror.Invalid("campo desconhecido: " + field)
	}
	return apperror.Invalid("JSON malformado")
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("id invalido")
	}
	return id, nil
}
func parseQuery(r *http.Request, allowed ...string) (url.Values, error) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, apperror.Invalid("query string malformada")
	}
	for _, name := range slices.Sorted(maps.Keys(values)) {
		if !slices.Contains(allowed, name) {
			return nil, apperror.Invalid("parametro desconhecido: " + name)
		}
		if len(values[name]) > 1 {
			return nil, apperror.Invalid("parametro repetido: " + name)
		}
	}
	return values, nil
}

func queryString(values url.Values, name string) *string {
	if !values.Has(name) {
		return nil
	}
	v := values.Get(name)
	return &v
}

func queryInt(values url.Values, name string) (*int, error) {
	s := queryString(values, name)
	if s == nil {
		return nil, nil
	}
	n, err := strconv.Atoi(*s)
	if err != nil {
		return nil, apperror.Invalid("parametro " + name + " deve ser um numero inteiro")
	}
	return &n, nil
}
