package domain

import "time"

// FilterByTimeRange returns entries within the [since, until] date range.
// Both boundaries are inclusive (until includes the entire end-of-day).
// Empty date strings mean no constraint on that boundary.
func FilterByTimeRange(entries []UsageEntry, since, until string, tz *time.Location) ([]UsageEntry, error) {
	if since == "" && until == "" {
		return entries, nil
	}

	var sinceTime, untilTime time.Time
	if since != "" {
		t, err := time.ParseInLocation("2006-01-02", since, tz)
		if err != nil {
			return nil, err
		}
		sinceTime = t
	}
	if until != "" {
		t, err := time.ParseInLocation("2006-01-02", until, tz)
		if err != nil {
			return nil, err
		}
		untilTime = t.Add(24*time.Hour - time.Nanosecond) // end of day
	}

	filtered := make([]UsageEntry, 0, len(entries))
	for _, e := range entries {
		local := e.Timestamp.In(tz)
		if !sinceTime.IsZero() && local.Before(sinceTime) {
			continue
		}
		if !untilTime.IsZero() && local.After(untilTime) {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered, nil
}
