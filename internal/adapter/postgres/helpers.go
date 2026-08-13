package postgres

import (
	"encoding/json"

	pid "github.com/stevenwilliam/healthy_catering/internal/platform/id"
)

// Small shared helpers, kept here so the repositories that need them do not
// each reach into platform/id with a different convention.

func orderCode() (string, error)    { return pid.OrderCode() }
func deliveryCode() (string, error) { return pid.Token(10) }
func uniqueSuffix() (int, error)    { return pid.UniqueCode() }

func addressSnapshotJSON(a Address) (string, error) {
	b, err := json.Marshal(addressSnapshot(a))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
