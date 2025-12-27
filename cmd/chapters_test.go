// Copyright © 2025 Kindly Ops, LLC <support@kindlyops.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/json"
	"testing"
)

func TestFFProbeResponse_Unmarshal(t *testing.T) {
	jsonData := `{
		"chapters": [
			{
				"id": 0,
				"time_base": "1/1000",
				"start": 0,
				"start_time": "0.000000",
				"end": 6006,
				"end_time": "6.006000",
				"tags": {
					"title": "Title Page"
				}
			},
			{
				"id": 1,
				"time_base": "1/1000",
				"start": 6006,
				"start_time": "6.006000",
				"end": 12012,
				"end_time": "12.012000",
				"tags": {
					"title": "Introduction"
				}
			}
		]
	}`

	var response ffmprobeResponse
	err := json.Unmarshal([]byte(jsonData), &response)

	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(response.Chapters) != 2 {
		t.Errorf("Expected 2 chapters, got %d", len(response.Chapters))
	}

	// Test first chapter
	firstChapter := response.Chapters[0]
	if firstChapter.ID != 0 {
		t.Errorf("Expected chapter ID 0, got %d", firstChapter.ID)
	}
	if firstChapter.StartTime != "0.000000" {
		t.Errorf("Expected start time '0.000000', got '%s'", firstChapter.StartTime)
	}
	if firstChapter.EndTime != "6.006000" {
		t.Errorf("Expected end time '6.006000', got '%s'", firstChapter.EndTime)
	}
	if firstChapter.Tags.Title != "Title Page" {
		t.Errorf("Expected title 'Title Page', got '%s'", firstChapter.Tags.Title)
	}

	// Test second chapter
	secondChapter := response.Chapters[1]
	if secondChapter.ID != 1 {
		t.Errorf("Expected chapter ID 1, got %d", secondChapter.ID)
	}
	if secondChapter.Tags.Title != "Introduction" {
		t.Errorf("Expected title 'Introduction', got '%s'", secondChapter.Tags.Title)
	}
}

func TestFFProbeResponse_UnmarshalWithCarriageReturn(t *testing.T) {
	// Test handling of titles with carriage returns (common in some video files)
	jsonData := `{
		"chapters": [
			{
				"id": 0,
				"start_time": "0.000000",
				"end_time": "6.006000",
				"tags": {
					"title": "Title Page\r"
				}
			}
		]
	}`

	var response ffmprobeResponse
	err := json.Unmarshal([]byte(jsonData), &response)

	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(response.Chapters) != 1 {
		t.Errorf("Expected 1 chapter, got %d", len(response.Chapters))
	}

	chapter := response.Chapters[0]
	// The title should include the \r as it's in the JSON
	if chapter.Tags.Title != "Title Page\r" {
		t.Errorf("Expected title 'Title Page\\r', got '%s'", chapter.Tags.Title)
	}
}

func TestFFProbeResponse_EmptyChapters(t *testing.T) {
	jsonData := `{"chapters": []}`

	var response ffmprobeResponse
	err := json.Unmarshal([]byte(jsonData), &response)

	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(response.Chapters) != 0 {
		t.Errorf("Expected 0 chapters, got %d", len(response.Chapters))
	}
}

func TestFFProbeResponse_Marshal(t *testing.T) {
	response := ffmprobeResponse{
		Chapters: []chapter{
			{
				ID:        0,
				StartTime: "0.000000",
				EndTime:   "5.000000",
				Tags: tags{
					Title: "Chapter 1",
				},
			},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Unmarshal it back to verify round-trip
	var decoded ffmprobeResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal marshaled data: %v", err)
	}

	if len(decoded.Chapters) != 1 {
		t.Errorf("Expected 1 chapter after round-trip, got %d", len(decoded.Chapters))
	}

	if decoded.Chapters[0].Tags.Title != "Chapter 1" {
		t.Errorf("Expected title 'Chapter 1' after round-trip, got '%s'", decoded.Chapters[0].Tags.Title)
	}
}

func TestChapter_DataStructure(t *testing.T) {
	// Test that chapter struct can hold expected values
	c := chapter{
		ID:        5,
		StartTime: "10.500000",
		EndTime:   "20.750000",
		Tags: tags{
			Title: "Test Chapter",
		},
	}

	if c.ID != 5 {
		t.Errorf("Expected ID 5, got %d", c.ID)
	}
	if c.StartTime != "10.500000" {
		t.Errorf("Expected start time '10.500000', got '%s'", c.StartTime)
	}
	if c.EndTime != "20.750000" {
		t.Errorf("Expected end time '20.750000', got '%s'", c.EndTime)
	}
	if c.Tags.Title != "Test Chapter" {
		t.Errorf("Expected title 'Test Chapter', got '%s'", c.Tags.Title)
	}
}
