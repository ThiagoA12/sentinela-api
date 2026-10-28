package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
)

const pgUniqueViolation = "23505"

func identifierInUse(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == "assets_identifier_active_key" {
		return apperror.Conflict("ja existe um ativo com este identificador", err)
	}
	return err
}
