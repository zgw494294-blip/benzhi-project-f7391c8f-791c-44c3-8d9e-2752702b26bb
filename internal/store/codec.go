package store

import (
	"encoding/json"
	"fmt"

	"seed-vigor-gate/internal/domain"
)

func encode(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode persistence value: %w", err)
	}
	return data, nil
}
func decodeCase(data []byte) (*domain.QualificationCase, error) {
	var item domain.QualificationCase
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("decode case aggregate: %w", err)
	}
	return &item, nil
}
func decodeCredential(data []byte) (*domain.EligibilityCredential, error) {
	var item domain.EligibilityCredential
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("decode credential: %w", err)
	}
	return &item, nil
}

func cloneCase(item *domain.QualificationCase) (*domain.QualificationCase, error) {
	data, err := encode(item)
	if err != nil {
		return nil, err
	}
	return decodeCase(data)
}
