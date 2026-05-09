package labextract

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ImagingSuggestion holds the Claude-suggested fields for an imaging study upload.
// Any field may be empty if the model couldn't determine it from the input.
type ImagingSuggestion struct {
	StudyType string `json:"study_type"`
	StudyDate string `json:"study_date"`
	Findings  string `json:"findings"`
	Notes     string `json:"notes"`
}

// ExtractImaging asks Claude Vision to suggest study_type, study_date, and findings
// for an imaging-study upload. Returns an empty suggestion when the input is not
// recognizable as a radiology report (vs. an error).
func (s *Service) ExtractImaging(ctx context.Context, images []ImageData) (*ImagingSuggestion, error) {
	raw, err := s.client.ExtractLabResults(ctx, images, ImagingExtractionPrompt())
	if err != nil {
		return nil, fmt.Errorf("claude API: %w", err)
	}
	return ParseImagingSuggestion(raw)
}

// ParseImagingSuggestion parses Claude's response into an ImagingSuggestion.
// Tolerates markdown code fences and missing fields.
func ParseImagingSuggestion(raw string) (*ImagingSuggestion, error) {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.SplitN(trimmed, "\n", 2)
		if len(lines) == 2 {
			trimmed = lines[1]
		}
		if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
			trimmed = trimmed[:idx]
		}
		trimmed = strings.TrimSpace(trimmed)
	}
	if trimmed == "" {
		return &ImagingSuggestion{}, nil
	}

	var s ImagingSuggestion
	if err := json.Unmarshal([]byte(trimmed), &s); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return &s, nil
}

// ImagingExtractionPrompt returns the system prompt for imaging-study extraction.
func ImagingExtractionPrompt() string {
	return `You are a medical imaging report assistant. Analyze the provided radiology image(s) or report page(s) and infer metadata.

Return ONLY a JSON object (no markdown, no explanation) with these fields:
- "study_type": The imaging modality. Use one of these canonical values when possible: "CT", "Ultrasound", "MRI", "X-ray", "HIDA". For other modalities, use the name as shown on the report. Empty string if unclear.
- "study_date": The study/exam/report date in YYYY-MM-DD format if visible. Empty string if not found.
- "findings": A concise extraction of the radiologist's findings or impression text from the report (1-3 sentences). Empty string if not present or unreadable.
- "notes": Optional free-text context (e.g., facility name, ordering physician). Empty string if nothing notable.

If the input is not a recognizable medical imaging report, return all fields as empty strings.

Example output:
{
  "study_type": "Ultrasound",
  "study_date": "2026-04-15",
  "findings": "Liver size at upper limit of normal. No focal lesions. Patent hepatic vasculature.",
  "notes": ""
}`
}
