package program

import (
	"fmt"
	"strings"

	"github.com/marianozunino/sdm-ui/internal/storage"
)

func parseDataSources(output string) []storage.DataSource {
	lines := extractSection(output, "DATASOURCE", "SERVER")
	var dataSources []storage.DataSource
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			statusIndex := -1
			for i, field := range fields {
				if field == "connected" {
					statusIndex = i
					break
				}
			}

			if statusIndex != -1 {
				dataSource := storage.DataSource{
					Name:    fields[0],
					Status:  strings.Join(fields[1:statusIndex+1], " "),
					Address: fields[statusIndex+1],
					Type:    fields[statusIndex+2],
					Tags:    strings.Join(fields[statusIndex+3:], " "),
				}
				if dataSource.Type == "amazones" {
					dataSource.Address = fmt.Sprintf("http://%s/_plugin/kibana/app/kibana", dataSource.Address)
				}
				dataSources = append(dataSources, dataSource)
			}
		}
	}
	return dataSources
}

func parseServers(output string) []storage.DataSource {
	lines := extractSection(output, "SERVER", "WEBSITE")
	var servers []storage.DataSource
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 5 {

			statusIndex := -1
			for i, field := range fields {
				if field == "connected" {
					statusIndex = i
					break
				}
			}
			if statusIndex != -1 {
				server := storage.DataSource{
					Name:    fields[0],
					Status:  strings.Join(fields[1:statusIndex+1], " "),
					Address: fmt.Sprintf("https://%s", fields[statusIndex+2]),
					Type:    fields[statusIndex+3],
					Tags:    strings.Join(fields[statusIndex+3:], " "),
				}
				servers = append(servers, server)
			}
		}
	}
	return servers
}

func extractSection(output, startMarker, endMarker string) []string {
	var lines []string
	start := strings.Index(output, startMarker)
	end := strings.Index(output, endMarker)
	if start == -1 {
		return lines
	}
	if end == -1 {
		end = len(output)
	}
	section := output[start:end]
	lines = strings.Split(section, "\n")
	// Remove the header line
	if len(lines) > 0 {
		lines = lines[1:]
	}
	// Remove empty lines
	var cleanedLines []string
	for _, line := range lines {
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}
	return cleanedLines
}
