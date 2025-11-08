package dynamodb

import (
	"testing"
	"time"

	"serverless-notification/domain/notification"
)

func TestToItem(t *testing.T) {
	// Arrange
	createdAt := time.Date(2024, 11, 3, 15, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 11, 3, 16, 0, 0, 0, time.UTC)

	notif := &notification.Notification{
		ID:          "01HQ8XA2B3C4D5E6F7G8H9",
		UserID:      "usr_123",
		Title:       "Test Notification",
		Content:     "This is a test",
		ChannelName: "email",
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	// Act
	item := toItem(notif)

	// Assert - Keys
	expectedPK := "USER#usr_123"
	if item.PK != expectedPK {
		t.Errorf("PK: expected %s, got %s", expectedPK, item.PK)
	}

	expectedSK := "NOTIF#2024-11-03T15:30:00Z#01HQ8XA2B3C4D5E6F7G8H9"
	if item.SK != expectedSK {
		t.Errorf("SK: expected %s, got %s", expectedSK, item.SK)
	}

	expectedGSI1PK := "NOTIF#01HQ8XA2B3C4D5E6F7G8H9"
	if item.GSI1PK != expectedGSI1PK {
		t.Errorf("GSI1PK: expected %s, got %s", expectedGSI1PK, item.GSI1PK)
	}

	expectedGSI1SK := "NOTIF#01HQ8XA2B3C4D5E6F7G8H9"
	if item.GSI1SK != expectedGSI1SK {
		t.Errorf("GSI1SK: expected %s, got %s", expectedGSI1SK, item.GSI1SK)
	}

	// Assert - Attributes
	if item.ID != notif.ID {
		t.Errorf("ID: expected %s, got %s", notif.ID, item.ID)
	}

	if item.UserID != notif.UserID {
		t.Errorf("UserID: expected %s, got %s", notif.UserID, item.UserID)
	}

	if item.Title != notif.Title {
		t.Errorf("Title: expected %s, got %s", notif.Title, item.Title)
	}

	if item.Content != notif.Content {
		t.Errorf("Content: expected %s, got %s", notif.Content, item.Content)
	}

	if item.ChannelName != notif.ChannelName {
		t.Errorf("ChannelName: expected %s, got %s", notif.ChannelName, item.ChannelName)
	}

	// Assert - Timestamps (ISO8601 format)
	expectedCreatedAt := "2024-11-03T15:30:00Z"
	if item.CreatedAt != expectedCreatedAt {
		t.Errorf("CreatedAt: expected %s, got %s", expectedCreatedAt, item.CreatedAt)
	}

	expectedUpdatedAt := "2024-11-03T16:00:00Z"
	if item.UpdatedAt != expectedUpdatedAt {
		t.Errorf("UpdatedAt: expected %s, got %s", expectedUpdatedAt, item.UpdatedAt)
	}
}

func TestToEntity(t *testing.T) {
	// Arrange
	item := NotificationItem{
		PK:          "USER#usr_123",
		SK:          "NOTIF#2024-11-03T15:30:00Z#01HQ8XA2B3C4D5E6F7G8H9",
		GSI1PK:      "NOTIF#01HQ8XA2B3C4D5E6F7G8H9",
		GSI1SK:      "NOTIF#01HQ8XA2B3C4D5E6F7G8H9",
		ID:          "01HQ8XA2B3C4D5E6F7G8H9",
		UserID:      "usr_123",
		Title:       "Test Notification",
		Content:     "This is a test",
		ChannelName: "email",
		CreatedAt:   "2024-11-03T15:30:00Z",
		UpdatedAt:   "2024-11-03T16:00:00Z",
	}

	// Act
	entity, err := toEntity(item)

	// Assert - No error
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Assert - Attributes
	if entity.ID != item.ID {
		t.Errorf("ID: expected %s, got %s", item.ID, entity.ID)
	}

	if entity.UserID != item.UserID {
		t.Errorf("UserID: expected %s, got %s", item.UserID, entity.UserID)
	}

	if entity.Title != item.Title {
		t.Errorf("Title: expected %s, got %s", item.Title, entity.Title)
	}

	if entity.Content != item.Content {
		t.Errorf("Content: expected %s, got %s", item.Content, entity.Content)
	}

	if entity.ChannelName != item.ChannelName {
		t.Errorf("ChannelName: expected %s, got %s", item.ChannelName, entity.ChannelName)
	}

	// Assert - Timestamps (parsed correctly)
	expectedCreatedAt := time.Date(2024, 11, 3, 15, 30, 0, 0, time.UTC)
	if !entity.CreatedAt.Equal(expectedCreatedAt) {
		t.Errorf("CreatedAt: expected %v, got %v", expectedCreatedAt, entity.CreatedAt)
	}

	expectedUpdatedAt := time.Date(2024, 11, 3, 16, 0, 0, 0, time.UTC)
	if !entity.UpdatedAt.Equal(expectedUpdatedAt) {
		t.Errorf("UpdatedAt: expected %v, got %v", expectedUpdatedAt, entity.UpdatedAt)
	}
}

func TestToEntity_InvalidTimestamp(t *testing.T) {
	// Arrange - Invalid created_at
	item := NotificationItem{
		ID:          "01HQ8XA2B3C4D5E6F7G8H9",
		UserID:      "usr_123",
		Title:       "Test",
		Content:     "Test",
		ChannelName: "email",
		CreatedAt:   "invalid-timestamp",
		UpdatedAt:   "2024-11-03T16:00:00Z",
	}

	// Act
	entity, err := toEntity(item)

	// Assert - Should return error
	if err == nil {
		t.Error("Expected error for invalid timestamp, got nil")
	}

	if entity != nil {
		t.Error("Expected nil entity on error")
	}
}

func TestToEntities(t *testing.T) {
	// Arrange
	items := []NotificationItem{
		{
			ID:          "01HQ8XA2B3C4D5E6F7G8H9",
			UserID:      "usr_123",
			Title:       "First",
			Content:     "First notification",
			ChannelName: "email",
			CreatedAt:   "2024-11-03T15:30:00Z",
			UpdatedAt:   "2024-11-03T15:30:00Z",
		},
		{
			ID:          "01HQ8XB2C3D4E5F6G7H8I9",
			UserID:      "usr_123",
			Title:       "Second",
			Content:     "Second notification",
			ChannelName: "sms",
			CreatedAt:   "2024-11-03T16:00:00Z",
			UpdatedAt:   "2024-11-03T16:00:00Z",
		},
	}

	// Act
	entities, err := toEntities(items)

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(entities) != 2 {
		t.Fatalf("Expected 2 entities, got %d", len(entities))
	}

	if entities[0].Title != "First" {
		t.Errorf("First entity title: expected 'First', got %s", entities[0].Title)
	}

	if entities[1].Title != "Second" {
		t.Errorf("Second entity title: expected 'Second', got %s", entities[1].Title)
	}
}

