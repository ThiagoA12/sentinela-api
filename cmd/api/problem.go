package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
)

type problem struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Detail   string         `json:"detail,omitempty"`
	Instance string         `json:"instance,omitempty"`
	Errors   []fieldProblem `json:"errors,omitempty"`
}

type fieldProblem struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func statusOf(k apperror.Kind) int {
	switch k {
	case apperror.KindInvalid:
		return http.StatusBadRequest
	case apperror.KindNotFound:
		return http.StatusNotFound
	case apperror.KindConflict:
		return http.StatusConflict
	case apperror.KindValidation:
		return http.StatusUnprocessableEntity
	case apperror.KindTooLarge:
		return http.StatusRequestEntityTooLarge
	case apperror.KindUnsupportedMedia:
		return http.StatusUnsupportedMediaType
	default:
		return http.StatusInternalServerError
	}
}

func writeProblem(w http.ResponseWriter, r *http.Request, err error) {
	p := problem{Type: "about:blank", Instance: r.URL.Path}

	var appErr *apperror.Error
	if errors.As(err, &appErr) && appErr.Kind != apperror.KindInternal {
		p.Status = statusOf(appErr.Kind)
		p.Detail = appErr.Detail
		for _, f := range appErr.Fields {
			p.Errors = append(p.Errors, fieldProblem{Field: f.Field, Reason: f.Reason})
		}
	} else {
		log.Printf("erro interno em %s %s: %v", r.Method, r.URL.Path, err)
		p.Status = http.StatusInternalServerError
		p.Detail = "erro inesperado. tente novamente mais tarde"
	}
	p.Title = http.StatusText(p.Status)

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		log.Printf("escrever ersposta de erro: %v", err)
	}
}
