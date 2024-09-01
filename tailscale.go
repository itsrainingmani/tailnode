package main

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/table"
)

func changeExitNode(ctx context.Context, exitNode string) error {
	cmd := exec.CommandContext(ctx, "tailscale", "set", "--exit-node="+exitNode)
	return cmd.Run()
}

func getCurrentExitNode() string {
	cmd := exec.Command("tailscale", "exit-node", "list")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "selected") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				country := fields[2]
				city := fields[3]
				if city == "Any" {
					continue
				}
				return country + ", " + city
			}
		}
	}

	return ""
}

func generateMullvadServers() ([]table.Row, error) {
	cmd := exec.Command("tailscale", "exit-node", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute tailscale command: %v", err)
	}

	var servers []table.Row
	lines := strings.Split(string(output), "\n")
	re := regexp.MustCompile(`\s{2,}`)

	for _, line := range lines[2:] { // Skip the header line
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := re.Split(strings.TrimSpace(line), -1)
		if len(fields) >= 5 {
			ip := fields[0]
			hostname := fields[1]
			country := fields[2]
			city := fields[3]
			if city == "Any" {
				continue
			}
			servers = append(servers, table.Row{ip, hostname, country, city})
		}
	}

	return servers, nil
}
