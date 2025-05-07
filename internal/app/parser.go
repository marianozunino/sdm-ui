package app

import (
	"encoding/json"

	"github.com/marianozunino/sdm-ui/internal/storage"
	"github.com/rs/zerolog/log"
)

// ResourceType represents the type of resource
type ResourceType string

// Resource type constants
const (
	TypeRedis        ResourceType = "redis"
	TypePostgres     ResourceType = "postgres"
	TypeAmazonEKS    ResourceType = "amazoneks"
	TypeAmazonES     ResourceType = "amazones"
	TypeAthena       ResourceType = "athena"
	TypeHTTPNoAuth   ResourceType = "httpNoAuth"
	TypeAmazonMQAMQP ResourceType = "amazonmq-amqp-091"
	TypeRawTCP       ResourceType = "rawtcp"
)

// Resource represents the details of a resource
type Resource struct {
	Address          string `json:"address,omitempty"`
	Connected        bool   `json:"connected"`
	ConnectionStatus string `json:"connection_status"`
	Hostname         string `json:"hostname"`
	ID               string `json:"id"`
	Message          string `json:"message"`
	Name             string `json:"name"`
	Tags             string `json:"tags"`
	Type             string `json:"type"`
	WebURL           string `json:"web_url,omitempty"`
}

// parseDataSources converts JSON-encoded resource data into a list of DataSource objects
func parseDataSources(rawResources string) []storage.DataSource {
	if rawResources == "" {
		log.Warn().Msg("Empty resource data received")
		return nil
	}

	log.Debug().
		Int("raw_length", len(rawResources)).
		Msg("Parsing resource data")

	var resources []Resource
	if err := json.Unmarshal([]byte(rawResources), &resources); err != nil {
		log.Error().
			Err(err).
			Str("raw_data_sample", truncateString(rawResources, 100)).
			Msg("Failed to parse resources JSON")
		return nil
	}

	log.Debug().
		Int("resource_count", len(resources)).
		Msg("Successfully parsed resources")

	// Pre-allocate dataSources slice
	dataSources := make([]storage.DataSource, 0, len(resources))

	for i, resource := range resources {
		dataSource := storage.DataSource{
			Name:    resource.Name,
			Status:  resource.ConnectionStatus,
			Type:    resource.Type,
			Tags:    resource.Tags,
			Address: resource.Address,
			WebURL:  resource.WebURL,
		}

		// Use Message as Address if Address is empty
		if resource.Address == "" {
			dataSource.Address = resource.Message
			log.Debug().
				Int("index", i).
				Str("name", resource.Name).
				Msg("Using message as address for resource")
		}

		dataSources = append(dataSources, dataSource)
	}

	log.Debug().
		Int("datasource_count", len(dataSources)).
		Msg("Created data sources from resources")

	return dataSources
}

// truncateString truncates a string to the specified length and adds "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
