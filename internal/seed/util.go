package seed

import (
	"fmt"

	"github.com/google/uuid"
)

func parseOrgID(id string) (uuid.UUID, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("organization id must be a valid UUID: %w", err)
	}
	return u, nil
}
