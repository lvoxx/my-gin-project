package service

import (
	"github.com/google/uuid"
)

// parseUUID is a small helper shared across all services to convert a string
// UUID into a typed uuid.UUID, returning a descriptive error on failure.
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
