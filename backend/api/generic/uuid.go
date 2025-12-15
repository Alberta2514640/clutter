package generic

import "github.com/google/uuid"

func IsValidUuid(id string) bool {
	u, err := uuid.Parse(id)
	if err != nil {
		return false
	}
	return u.Version() == 4
}
