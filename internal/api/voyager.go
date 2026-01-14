package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// VoyagerResponse wraps LinkedIn's Voyager API response format.
type VoyagerResponse struct {
	Data     json.RawMessage `json:"data"`
	Included []json.RawMessage `json:"included"`
	Paging   *Paging           `json:"paging,omitempty"`
}

// Paging contains pagination information.
type Paging struct {
	Count int    `json:"count"`
	Start int    `json:"start"`
	Total int    `json:"total,omitempty"`
	Links []Link `json:"links,omitempty"`
}

// Link represents a pagination link.
type Link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
	Type string `json:"type"`
}

// ProfileResponse represents the profile API response.
type ProfileResponse struct {
	Profile   *Profile `json:"profile"`
	RawData   json.RawMessage
	RawIncluded []json.RawMessage
}

// GetMyProfile fetches the authenticated user's profile.
func (c *Client) GetMyProfile(ctx context.Context) (*Profile, error) {
	// Use the /me endpoint to get current user.
	var result VoyagerResponse
	err := c.Get(ctx, "/identity/dash/profiles?q=memberIdentity&memberIdentity=me&decorationId=com.linkedin.voyager.dash.deco.identity.profile.WebTopCardCore-19", nil, &result)
	if err != nil {
		return nil, err
	}

	return parseProfileFromResponse(&result)
}

// GetProfile fetches a profile by public identifier (username).
func (c *Client) GetProfile(ctx context.Context, publicID string) (*Profile, error) {
	// URL encode the public ID.
	encodedID := url.PathEscape(publicID)

	path := fmt.Sprintf("/identity/profiles/%s/profileView", encodedID)
	var result VoyagerResponse
	if err := c.Get(ctx, path, nil, &result); err != nil {
		return nil, err
	}

	return parseProfileFromResponse(&result)
}

// GetProfileByURN fetches a profile by URN.
func (c *Client) GetProfileByURN(ctx context.Context, urn string) (*Profile, error) {
	// Extract the member ID from URN.
	// URN format: urn:li:member:123456 or urn:li:fsd_profile:ACoAAAxxxxxx
	parts := strings.Split(urn, ":")
	if len(parts) < 4 {
		return nil, &Error{
			Code:    ErrCodeInvalidInput,
			Message: fmt.Sprintf("invalid URN format: %s", urn),
		}
	}

	memberID := parts[len(parts)-1]

	// Use the profile API with URN.
	query := url.Values{}
	query.Set("q", "memberIdentity")
	query.Set("memberIdentity", memberID)
	query.Set("decorationId", "com.linkedin.voyager.dash.deco.identity.profile.WebTopCardCore-19")

	var result VoyagerResponse
	if err := c.Get(ctx, "/identity/dash/profiles", query, &result); err != nil {
		return nil, err
	}

	return parseProfileFromResponse(&result)
}

// parseProfileFromResponse extracts a Profile from a Voyager response.
func parseProfileFromResponse(resp *VoyagerResponse) (*Profile, error) {
	if resp == nil {
		return nil, &Error{
			Code:    ErrCodeServerError,
			Message: "empty response",
		}
	}

	// The profile data can be in different places depending on the endpoint.
	// Try to find it in the included array first.
	profile := &Profile{}

	for _, item := range resp.Included {
		var entity map[string]json.RawMessage
		if err := json.Unmarshal(item, &entity); err != nil {
			continue
		}

		// Check for profile entity.
		if entityURN, ok := entity["entityUrn"]; ok {
			var urn string
			if err := json.Unmarshal(entityURN, &urn); err == nil {
				if strings.Contains(urn, "fsd_profile") || strings.Contains(urn, "member") {
					if err := parseProfileEntity(item, profile); err == nil {
						if profile.FirstName != "" || profile.PublicID != "" {
							return profile, nil
						}
					}
				}
			}
		}

		// Check for $type field.
		if typeField, ok := entity["$type"]; ok {
			var typeName string
			if err := json.Unmarshal(typeField, &typeName); err == nil {
				if strings.Contains(typeName, "Profile") || strings.Contains(typeName, "MiniProfile") {
					if err := parseProfileEntity(item, profile); err == nil {
						if profile.FirstName != "" || profile.PublicID != "" {
							return profile, nil
						}
					}
				}
			}
		}
	}

	// Try parsing the data field directly.
	if len(resp.Data) > 0 {
		if err := parseProfileEntity(resp.Data, profile); err == nil {
			if profile.FirstName != "" || profile.PublicID != "" {
				return profile, nil
			}
		}
	}

	// If we got here with some data, return what we have.
	if profile.URN != "" || profile.FirstName != "" || profile.PublicID != "" {
		return profile, nil
	}

	return nil, &Error{
		Code:    ErrCodeNotFound,
		Message: "profile not found in response",
	}
}

// parseProfileEntity extracts profile fields from a JSON entity.
func parseProfileEntity(data json.RawMessage, profile *Profile) error {
	var entity struct {
		EntityURN      string `json:"entityUrn"`
		PublicIdentifier string `json:"publicIdentifier"`
		FirstName      string `json:"firstName"`
		LastName       string `json:"lastName"`
		Headline       string `json:"headline"`
		Summary        string `json:"summary"`
		LocationName   string `json:"locationName"`
		GeoLocationName string `json:"geoLocationName"`
		ProfilePicture struct {
			DisplayImageReference struct {
				VectorImage struct {
					RootURL string `json:"rootUrl"`
				} `json:"vectorImage"`
			} `json:"displayImageReference"`
		} `json:"profilePicture"`
		// Alternative field names.
		Occupation string `json:"occupation"`
		MiniProfile struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			Occupation string `json:"occupation"`
			PublicIdentifier string `json:"publicIdentifier"`
			EntityUrn string `json:"entityUrn"`
		} `json:"miniProfile"`
	}

	if err := json.Unmarshal(data, &entity); err != nil {
		return err
	}

	// Set fields from direct properties.
	if entity.EntityURN != "" {
		profile.URN = entity.EntityURN
	}
	if entity.PublicIdentifier != "" {
		profile.PublicID = entity.PublicIdentifier
		profile.ProfileURL = fmt.Sprintf("https://www.linkedin.com/in/%s", entity.PublicIdentifier)
	}
	if entity.FirstName != "" {
		profile.FirstName = entity.FirstName
	}
	if entity.LastName != "" {
		profile.LastName = entity.LastName
	}
	if entity.Headline != "" {
		profile.Headline = entity.Headline
	} else if entity.Occupation != "" {
		profile.Headline = entity.Occupation
	}
	if entity.Summary != "" {
		profile.Summary = entity.Summary
	}
	if entity.LocationName != "" {
		profile.Location = entity.LocationName
	} else if entity.GeoLocationName != "" {
		profile.Location = entity.GeoLocationName
	}
	if entity.ProfilePicture.DisplayImageReference.VectorImage.RootURL != "" {
		profile.ProfilePicURL = entity.ProfilePicture.DisplayImageReference.VectorImage.RootURL
	}

	// Set fields from miniProfile if main fields are empty.
	if profile.FirstName == "" && entity.MiniProfile.FirstName != "" {
		profile.FirstName = entity.MiniProfile.FirstName
	}
	if profile.LastName == "" && entity.MiniProfile.LastName != "" {
		profile.LastName = entity.MiniProfile.LastName
	}
	if profile.Headline == "" && entity.MiniProfile.Occupation != "" {
		profile.Headline = entity.MiniProfile.Occupation
	}
	if profile.PublicID == "" && entity.MiniProfile.PublicIdentifier != "" {
		profile.PublicID = entity.MiniProfile.PublicIdentifier
		profile.ProfileURL = fmt.Sprintf("https://www.linkedin.com/in/%s", entity.MiniProfile.PublicIdentifier)
	}
	if profile.URN == "" && entity.MiniProfile.EntityUrn != "" {
		profile.URN = entity.MiniProfile.EntityUrn
	}

	return nil
}

// FeedOptions configures feed fetching.
type FeedOptions struct {
	Limit int
	Start int
}

// GetFeed fetches the user's LinkedIn feed.
func (c *Client) GetFeed(ctx context.Context, opts *FeedOptions) ([]FeedItem, error) {
	if opts == nil {
		opts = &FeedOptions{Limit: 10}
	}
	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	query := url.Values{}
	query.Set("count", fmt.Sprintf("%d", opts.Limit))
	query.Set("start", fmt.Sprintf("%d", opts.Start))
	query.Set("q", "feedByType")
	query.Set("feedType", "HOMEPAGE")

	var result VoyagerResponse
	if err := c.Get(ctx, "/feed/updatesV2", query, &result); err != nil {
		return nil, err
	}

	return parseFeedFromResponse(&result)
}

// parseFeedFromResponse extracts feed items from a Voyager response.
func parseFeedFromResponse(resp *VoyagerResponse) ([]FeedItem, error) {
	if resp == nil {
		return nil, &Error{
			Code:    ErrCodeServerError,
			Message: "empty response",
		}
	}

	var items []FeedItem

	// Feed items are typically in the included array.
	for _, raw := range resp.Included {
		var entity map[string]json.RawMessage
		if err := json.Unmarshal(raw, &entity); err != nil {
			continue
		}

		// Look for update entities.
		if typeField, ok := entity["$type"]; ok {
			var typeName string
			if err := json.Unmarshal(typeField, &typeName); err == nil {
				if strings.Contains(typeName, "Update") || strings.Contains(typeName, "Activity") {
					item, err := parseFeedItem(raw)
					if err == nil && item != nil {
						items = append(items, *item)
					}
				}
			}
		}
	}

	return items, nil
}

// parseFeedItem parses a single feed item.
func parseFeedItem(data json.RawMessage) (*FeedItem, error) {
	var entity struct {
		EntityURN string `json:"entityUrn"`
		Actor     struct {
			URN  string `json:"urn"`
			Name struct {
				Text string `json:"text"`
			} `json:"name"`
		} `json:"actor"`
		Commentary struct {
			Text struct {
				Text string `json:"text"`
			} `json:"text"`
		} `json:"commentary"`
		SocialDetail struct {
			URN          string `json:"urn"`
			TotalLikes   int    `json:"totalSocialActivityCounts,omitempty"`
			LikesCount   int    `json:"likes,omitempty"`
			CommentsCount int   `json:"comments,omitempty"`
		} `json:"socialDetail"`
		CreatedAt int64 `json:"createdAt"`
	}

	if err := json.Unmarshal(data, &entity); err != nil {
		return nil, err
	}

	if entity.EntityURN == "" {
		return nil, fmt.Errorf("no URN in feed item")
	}

	item := &FeedItem{
		URN:  entity.EntityURN,
		Type: "update",
	}

	if entity.Commentary.Text.Text != "" {
		item.Post = &Post{
			URN:  entity.EntityURN,
			Text: entity.Commentary.Text.Text,
		}
	}

	if entity.Actor.Name.Text != "" {
		item.Actor = &Profile{
			URN:       entity.Actor.URN,
			FirstName: entity.Actor.Name.Text,
		}
	}

	return item, nil
}

// CreatePost creates a new LinkedIn post.
func (c *Client) CreatePost(ctx context.Context, text string) (*Post, error) {
	// First get the current user's URN.
	profile, err := c.GetMyProfile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile for posting: %w", err)
	}

	authorURN := profile.URN
	if authorURN == "" {
		return nil, &Error{
			Code:    ErrCodeServerError,
			Message: "could not determine author URN",
		}
	}

	// Create post payload.
	payload := map[string]any{
		"author":               authorURN,
		"lifecycleState":       "PUBLISHED",
		"specificContent": map[string]any{
			"com.linkedin.ugc.ShareContent": map[string]any{
				"shareCommentary": map[string]any{
					"text": text,
				},
				"shareMediaCategory": "NONE",
			},
		},
		"visibility": map[string]any{
			"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC",
		},
	}

	var result map[string]any
	if err := c.Post(ctx, "/ugcPosts", payload, &result); err != nil {
		return nil, err
	}

	// Extract the created post URN.
	postURN := ""
	if id, ok := result["id"].(string); ok {
		postURN = id
	}

	return &Post{
		URN:       postURN,
		AuthorURN: authorURN,
		Text:      text,
	}, nil
}

// GetPost fetches a post by URN.
func (c *Client) GetPost(ctx context.Context, urn string) (*Post, error) {
	// URL encode the URN.
	encodedURN := url.PathEscape(urn)

	var result VoyagerResponse
	if err := c.Get(ctx, "/feed/updates/"+encodedURN, nil, &result); err != nil {
		return nil, err
	}

	// Parse the post from response.
	for _, raw := range result.Included {
		item, err := parseFeedItem(raw)
		if err == nil && item != nil && item.Post != nil {
			return item.Post, nil
		}
	}

	return nil, &Error{
		Code:    ErrCodeNotFound,
		Message: "post not found",
	}
}
