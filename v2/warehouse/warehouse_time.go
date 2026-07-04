package warehouse

import "time"

// Warehouse open/close/close-order are stored as time-of-day; the proto carries them
// as "HH:MM" strings (empty = unset), matching db_models.Warehouse's JSON marshaling.

func parseHHMM(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("15:04", s)
	if err != nil {
		return nil
	}
	return &t
}

func formatHHMM(t *time.Time) string {
	if t == nil {
		return ""
	}
	// Times are parsed/stored as UTC; normalize back to UTC before formatting so the
	// round-trip through a TIMESTAMPTZ column (which the driver localizes on read) is stable.
	return t.UTC().Format("15:04")
}
