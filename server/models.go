package server

import "encoding/json"

type Campaign struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Slug       string  `json:"slug"`
	Status     string  `json:"status"`
	StartsAt   *string `json:"startsAt"`
	EndsAt     *string `json:"endsAt"`
	FormLocked bool    `json:"formLocked"`
}
type Field struct {
	ID          int64          `json:"id"`
	CampaignID  int64          `json:"campaignId,omitempty"`
	Key         string         `json:"key"`
	Label       string         `json:"label"`
	Type        string         `json:"type"`
	Required    bool           `json:"required"`
	Placeholder string         `json:"placeholder"`
	HelpText    string         `json:"helpText"`
	Options     []string       `json:"options"`
	Position    int            `json:"position"`
	Validation  map[string]any `json:"validation"`
}
type ReviewStatus struct {
	ID          int64  `json:"id"`
	CampaignID  int64  `json:"campaignId,omitempty"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	Position    int    `json:"position"`
	IsDefault   bool   `json:"isDefault"`
}
type ApplicationSummary struct {
	ID           int64         `json:"id"`
	CampaignID   int64         `json:"campaignId"`
	StudentID    string        `json:"studentId"`
	Email        string        `json:"email"`
	SystemStatus string        `json:"systemStatus"`
	Revision     int           `json:"revision"`
	SubmittedAt  string        `json:"submittedAt"`
	ReviewStatus *ReviewStatus `json:"reviewStatus,omitempty"`
}

func decodeStrings(v string) []string {
	var out []string
	_ = json.Unmarshal([]byte(v), &out)
	if out == nil {
		out = []string{}
	}
	return out
}
func decodeMap(v string) map[string]any {
	out := map[string]any{}
	_ = json.Unmarshal([]byte(v), &out)
	return out
}
