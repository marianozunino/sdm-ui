package program

import (
	"encoding/json"

	"github.com/marianozunino/sdm-ui/internal/storage"
	"github.com/rs/zerolog/log"
)

// ResourceType represents the type of resource.
type ResourceType string

// Resource type constants.
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

// Resource represents the details of a resource.
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

// parseDataSources converts JSON-encoded resource data into a list of DataSource objects.
func parseDataSources(rawResources string) []storage.DataSource {
	var resources []Resource
	if err := json.Unmarshal([]byte(rawResources), &resources); err != nil {
		log.Debug().Msgf("Failed to parse resources: %s", err)
		return nil
	}

	var dataSources []storage.DataSource
	for _, resource := range resources {
		dataSource := storage.DataSource{
			Name:    resource.Name,
			Status:  resource.ConnectionStatus,
			Type:    resource.Type,
			Tags:    resource.Tags,
			Address: resource.Address,
			WebURL:  resource.WebURL,
		}

		if resource.Address == "" {
			dataSource.Address = resource.Message
		}

		dataSources = append(dataSources, dataSource)
	}

	return dataSources
}
