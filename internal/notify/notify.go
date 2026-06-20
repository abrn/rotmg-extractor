// Package notify delivers new-build notifications. Currently it implements a
// Discord webhook. It is decoupled from the diff packages: callers pass plain
// counts, not diff types.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"rotmg-extractor/internal/logx"
)

// Notification is the data shown in a new-build message.
type Notification struct {
	Env       string
	BuildType string
	Version   string
	Hash      string

	// File-tree diff counts.
	NewFiles     int
	DelFiles     int
	AddedLines   int
	RemovedLines int

	// Semantic game-data diff counts.
	ObjAdded, ObjRemoved, ObjChanged int
	GndAdded, GndRemoved, GndChanged int
}

// Notifier delivers a Notification.
type Notifier interface {
	Notify(ctx context.Context, n Notification) error
}

// Discord posts a rich embed to a Discord webhook URL.
type Discord struct {
	WebhookURL string
	RoleID     string // optional role to ping
	Log        *logx.Logger
}

// Notify sends the notification to Discord.
func (d *Discord) Notify(ctx context.Context, n Notification) error {
	payload := buildPayload(n, d.RoleID)
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned %s", resp.Status)
	}
	if d.Log != nil {
		d.Log.Info("Sent Discord notification")
	}
	return nil
}

// buildPayload constructs the Discord webhook JSON. Exposed package-internal for
// testing.
func buildPayload(n Notification, roleID string) map[string]any {
	content := fmt.Sprintf("A new RotMG %s %s build is available", n.Env, n.BuildType)
	if roleID != "" {
		content += fmt.Sprintf(" <@&%s>", roleID)
	}

	version := n.Version
	if version == "" {
		version = "unknown"
	}

	fields := []map[string]any{
		{"name": "Environment", "value": n.Env, "inline": true},
		{"name": "Type", "value": n.BuildType, "inline": true},
		{"name": "Version", "value": version, "inline": true},
		{"name": "Build Hash", "value": "`" + n.Hash + "`", "inline": false},
		{"name": "Objects", "value": fmt.Sprintf("`+%d -%d ~%d`", n.ObjAdded, n.ObjRemoved, n.ObjChanged), "inline": true},
		{"name": "Ground", "value": fmt.Sprintf("`+%d -%d ~%d`", n.GndAdded, n.GndRemoved, n.GndChanged), "inline": true},
		{"name": "Files / Lines", "value": fmt.Sprintf("```diff\nfiles: +%d -%d\nlines: +%d -%d\n```", n.NewFiles, n.DelFiles, n.AddedLines, n.RemovedLines), "inline": false},
	}

	return map[string]any{
		"content": content,
		"embeds": []map[string]any{{
			"title":  fmt.Sprintf("%s %s — %s", n.Env, n.BuildType, version),
			"color":  3368617, // #336699-ish blue
			"fields": fields,
		}},
	}
}
