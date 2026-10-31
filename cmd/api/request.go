package main

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
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
