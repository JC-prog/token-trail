package collector

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jcprog/token-trail/internal/database"
	"github.com/jcprog/token-trail/internal/models"
	"github.com/jcprog/token-trail/internal/pricing"
)

type ImportResult struct {
	RowsImported int     `json:"rows_imported"`
	RowsSkipped  int     `json:"rows_skipped"`
	Errors       []string `json:"errors"`
}

type Importer struct {
	db *database.DB
}

// NewImporter creates a new importer
func NewImporter(db *database.DB) *Importer {
	return &Importer{db: db}
}

// ImportCSV imports usage events from a CSV file
func (imp *Importer) ImportCSV(filepath string) (*ImportResult, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	result := &ImportResult{}

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Find column indices
	columnMap := make(map[string]int)
	for i, col := range header {
		columnMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Read rows
	lineNum := 2 // Skip header
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: %v", lineNum, err))
			result.RowsSkipped++
			lineNum++
			continue
		}

		// Parse event
		event, err := imp.parseCSVRow(record, columnMap)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: %v", lineNum, err))
			result.RowsSkipped++
			lineNum++
			continue
		}

		// Insert event
		if err := imp.db.InsertUsageEvent(event); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: failed to insert: %v", lineNum, err))
			result.RowsSkipped++
			lineNum++
			continue
		}

		result.RowsImported++
		lineNum++
	}

	return result, nil
}

// ImportJSON imports usage events from a JSON file
func (imp *Importer) ImportJSON(filepath string) (*ImportResult, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var jsonData []map[string]interface{}
	if err := json.NewDecoder(file).Decode(&jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	result := &ImportResult{}

	// Read rows
	for i, row := range jsonData {
		event, err := imp.parseJSONRow(row)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", i, err))
			result.RowsSkipped++
			continue
		}

		// Insert event
		if err := imp.db.InsertUsageEvent(event); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: failed to insert: %v", i, err))
			result.RowsSkipped++
			continue
		}

		result.RowsImported++
	}

	return result, nil
}

// parseCSVRow parses a CSV row into a UsageEvent
func (imp *Importer) parseCSVRow(record []string, columnMap map[string]int) (*models.UsageEvent, error) {
	event := &models.UsageEvent{
		ID:     uuid.New().String(),
		Source: "manual_import",
		Metadata: "{}",
	}

	// Provider
	if idx, ok := columnMap["provider"]; ok && idx < len(record) {
		event.ProviderID = strings.TrimSpace(record[idx])
	}

	// Model
	if idx, ok := columnMap["model"]; ok && idx < len(record) {
		event.Model = strings.TrimSpace(record[idx])
	}

	// Input tokens
	if idx, ok := columnMap["input_tokens"]; ok && idx < len(record) {
		val, err := strconv.Atoi(strings.TrimSpace(record[idx]))
		if err != nil {
			return nil, fmt.Errorf("invalid input_tokens: %w", err)
		}
		event.InputTokens = val
	}

	// Output tokens
	if idx, ok := columnMap["output_tokens"]; ok && idx < len(record) {
		val, err := strconv.Atoi(strings.TrimSpace(record[idx]))
		if err != nil {
			return nil, fmt.Errorf("invalid output_tokens: %w", err)
		}
		event.OutputTokens = val
	}

	// Timestamp
	if idx, ok := columnMap["timestamp"]; ok && idx < len(record) {
		val := strings.TrimSpace(record[idx])
		t, err := time.Parse(time.RFC3339, val)
		if err != nil {
			// Try alternate format
			t, err = time.Parse("2006-01-02 15:04:05", val)
			if err != nil {
				return nil, fmt.Errorf("invalid timestamp: %w", err)
			}
		}
		event.Timestamp = t
	}

	// Validate required fields
	if event.ProviderID == "" || event.Model == "" || event.Timestamp.IsZero() {
		return nil, fmt.Errorf("missing required fields: provider, model, timestamp")
	}

	// Generate dedup hash
	event.DedupHash = pricing.GenerateDedupHash(event.ProviderID, event.Model, event.Timestamp, event.InputTokens, event.OutputTokens)

	return event, nil
}

// parseJSONRow parses a JSON row into a UsageEvent
func (imp *Importer) parseJSONRow(row map[string]interface{}) (*models.UsageEvent, error) {
	event := &models.UsageEvent{
		ID:     uuid.New().String(),
		Source: "manual_import",
		Metadata: "{}",
	}

	// Provider
	if v, ok := row["provider"]; ok {
		event.ProviderID = fmt.Sprint(v)
	}

	// Model
	if v, ok := row["model"]; ok {
		event.Model = fmt.Sprint(v)
	}

	// Input tokens
	if v, ok := row["input_tokens"]; ok {
		switch val := v.(type) {
		case float64:
			event.InputTokens = int(val)
		case string:
			i, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid input_tokens: %w", err)
			}
			event.InputTokens = i
		}
	}

	// Output tokens
	if v, ok := row["output_tokens"]; ok {
		switch val := v.(type) {
		case float64:
			event.OutputTokens = int(val)
		case string:
			i, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid output_tokens: %w", err)
			}
			event.OutputTokens = i
		}
	}

	// Timestamp
	if v, ok := row["timestamp"]; ok {
		var t time.Time
		var err error

		switch val := v.(type) {
		case string:
			t, err = time.Parse(time.RFC3339, val)
			if err != nil {
				t, err = time.Parse("2006-01-02 15:04:05", val)
			}
		case float64:
			t = time.Unix(int64(val), 0)
		}

		if err != nil || t.IsZero() {
			return nil, fmt.Errorf("invalid timestamp: %w", err)
		}

		event.Timestamp = t
	}

	// Validate required fields
	if event.ProviderID == "" || event.Model == "" || event.Timestamp.IsZero() {
		return nil, fmt.Errorf("missing required fields: provider, model, timestamp")
	}

	// Generate dedup hash
	event.DedupHash = pricing.GenerateDedupHash(event.ProviderID, event.Model, event.Timestamp, event.InputTokens, event.OutputTokens)

	return event, nil
}
