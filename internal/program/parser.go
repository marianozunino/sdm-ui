package program

import (
	"encoding/json"
	"fmt"

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
		if !isValidType(resource.Type) {
			log.Debug().Msgf("Skipping invalid resource type: %s", resource.Type)
			continue
		}

		dataSource := storage.DataSource{
			Name:    resource.Name,
			Status:  resource.ConnectionStatus,
			Type:    resource.Type,
			Tags:    resource.Tags,
			Address: formatAddress(resource.Type, resource.Message),
		}
		dataSources = append(dataSources, dataSource)
	}

	return dataSources
}

// isValidType checks if the resource type is one of the recognized types.
func isValidType(resourceType string) bool {
	switch ResourceType(resourceType) {
	case TypeRedis, TypePostgres, TypeAmazonEKS, TypeAmazonES, TypeAthena, TypeAmazonMQAMQP, TypeRawTCP:
		return true
	default:
		return false
	}
}

// formatAddress formats the address for certain resource types.
func formatAddress(resourceType string, message string) string {
	switch ResourceType(resourceType) {
	case TypeAmazonES:
		return fmt.Sprintf("http://%s/_plugin/kibana/app/kibana", message)
	case TypeRawTCP:
		return fmt.Sprintf("https://%s", message)
	default:
		return message
	}
}

