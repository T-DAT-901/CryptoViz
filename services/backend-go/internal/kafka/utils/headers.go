package utils

import (
	"fmt"
	"strconv"
)

// MessageHeader represents a generic Kafka message header
type MessageHeader struct {
	Key   string
	Value []byte
}

// MessageMetadata contient les métadonnées extraites des headers Kafka
type MessageMetadata struct {
	Source    string
	Type      string
	Exchange  string
	Symbol    string
	Schema    string
	Timeframe string
	Closed    bool
}

// ParseHeaders extrait les métadonnées des headers Kafka
func ParseHeaders(headers []MessageHeader) (*MessageMetadata, error) {
	metadata := &MessageMetadata{}

	for _, header := range headers {
		key := header.Key
		value := string(header.Value)

		switch key {
		case "source":
			metadata.Source = value
		case "type":
			metadata.Type = value
		case "exchange":
			metadata.Exchange = value
		case "symbol":
			metadata.Symbol = value
		case "schema":
			metadata.Schema = value
		case "timeframe":
			metadata.Timeframe = value
		case "closed":
			closed, err := strconv.ParseBool(value)
			if err != nil {
				metadata.Closed = false
			} else {
				metadata.Closed = closed
			}
		}
	}

	return metadata, nil
}

// GetHeaderValue récupère la valeur d'un header spécifique
func GetHeaderValue(headers []MessageHeader, key string) (string, bool) {
	for _, header := range headers {
		if header.Key == key {
			return string(header.Value), true
		}
	}
	return "", false
}

// GetHeaderBool récupère une valeur booléenne d'un header
func GetHeaderBool(headers []MessageHeader, key string) (bool, error) {
	value, found := GetHeaderValue(headers, key)
	if !found {
		return false, fmt.Errorf("header %s not found", key)
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("failed to parse header %s as bool: %w", key, err)
	}

	return boolValue, nil
}

// ValidateHeaders vérifie que les headers requis sont présents
func ValidateHeaders(headers []MessageHeader, requiredKeys []string) error {
	headerMap := make(map[string]bool)
	for _, header := range headers {
		headerMap[header.Key] = true
	}

	for _, key := range requiredKeys {
		if !headerMap[key] {
			return fmt.Errorf("required header %s is missing", key)
		}
	}

	return nil
}
