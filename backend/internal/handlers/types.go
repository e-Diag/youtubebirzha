package handlers

import (
	"fmt"
	"time"

	"youtube-market/internal/models"
)

type AdView struct {
	ID         uint      `json:"id"`
	Username   string    `json:"username"`
	Title      string    `json:"title"`
	Desc       string    `json:"desc"`
	Category   string    `json:"category"`
	Mode       string    `json:"mode"`
	Tag        string    `json:"tag"`
	IsPremium  bool      `json:"is_premium"`
	Status     string    `json:"status"`
	ExpiresAt  time.Time `json:"expires_at"`
	PhotoURL   string    `json:"photo_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func buildAdView(ad models.Ad) AdView {
	view := AdView{
		ID:        ad.ID,
		Username:  ad.Username,
		Title:     ad.Title,
		Desc:      ad.Desc,
		Category:  ad.Category,
		Mode:      ad.Mode,
		Tag:       ad.Tag,
		IsPremium: ad.IsPremium,
		Status:    ad.Status,
		ExpiresAt: ad.ExpiresAt,
		CreatedAt: ad.CreatedAt,
		UpdatedAt: ad.UpdatedAt,
	}

	if ad.PhotoPath != "" {
		view.PhotoURL = fmt.Sprintf("/api/ads/%d/photo", ad.ID)
	}

	return view
}

func mergeAds(primary, secondary []models.Ad) []models.Ad {
	if len(primary) == 0 {
		return secondary
	}

	seen := make(map[uint]struct{}, len(primary)+len(secondary))
	out := make([]models.Ad, 0, len(primary)+len(secondary))

	for _, ad := range primary {
		if _, ok := seen[ad.ID]; ok {
			continue
		}
		out = append(out, ad)
		seen[ad.ID] = struct{}{}
	}

	for _, ad := range secondary {
		if _, ok := seen[ad.ID]; ok {
			continue
		}
		out = append(out, ad)
		seen[ad.ID] = struct{}{}
	}

	return out
}

