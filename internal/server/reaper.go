package server

import (
	"context"
	"log/slog"
	"time"
)

// StartReaper launches a background goroutine that permanently deletes user accounts
// that have been deactivated for 30 or more days. It ticks once every 24 hours.
func (cfg *ApiConfig) StartReaper() {
	go func() {
		ticker := time.NewTicker(cfg.ReaperInterval)
		defer ticker.Stop()

		for range ticker.C {
			count, err := cfg.DB.ReapDeactivatedUsers(context.Background())
			if err != nil {
				slog.Error("Reaper failed to delete deactivated accounts", "error", err)
				continue
			}
			if count > 0 {
				slog.Info("Reaper deleted deactivated accounts", "count", count)
			}
		}
	}()
}
