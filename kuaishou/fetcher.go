package kuaishou

import (
	"bytes"
	"errors"
	"github.com/goccy/go-json"
	"io"
	"net/http"
)

// UserProfile structure to unmarshal JSON response
type UserProfile struct {
	Data struct {
		VisionProfile struct {
			Result      int    `json:"result"`
			HostName    string `json:"hostName"`
			UserProfile struct {
				OwnerCount interface{} `json:"ownerCount"`
				Profile    struct {
					Gender           string `json:"gender"`
					UserName         string `json:"user_name"`
					UserID           string `json:"user_id"`
					HeadURL          string `json:"headurl"`
					UserText         string `json:"user_text"`
					UserProfileBgURL string `json:"user_profile_bg_url"`
					Typename         string `json:"__typename"`
				} `json:"profile"`
				IsFollowing bool   `json:"isFollowing"`
				Typename    string `json:"__typename"`
			} `json:"userProfile"`
			Typename string `json:"__typename"`
		} `json:"visionProfile"`
	} `json:"data"`
}

func GetUserInfo(principalId string, cookie string) (*UserProfile, error) {
	if cookie == "" {
		return nil, errors.New("cookie is empty")
	}

	// Set up the payload
	payload := map[string]interface{}{
		"operationName": "visionProfile",
		"variables": map[string]string{
			"userId": principalId,
		},
		"query": `query visionProfile($userId: String) { visionProfile(userId: $userId) { result hostName userProfile { ownerCount { fan photo follow photo_public __typename } profile { gender user_name user_id headurl user_text user_profile_bg_url __typename } isFollowing __typename } __typename } }`,
	}

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Create a new POST request
	req, err := http.NewRequest("POST", "https://www.kuaishou.com/graphql", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/50.0.2661.87 Safari/537.36")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON response
	var userProfile UserProfile
	err = json.Unmarshal(body, &userProfile)
	if err != nil {
		return nil, err
	}

	return &userProfile, nil
}
