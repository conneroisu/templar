package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validatePluginsConfig validates plugins configuration values
func validatePluginsConfig(config *PluginsConfig) error {
	// Validate discovery paths
	for _, path := range config.DiscoveryPaths {
		// Clean the path
		cleanPath := filepath.Clean(path)
		
		// Reject path traversal attempts
		if strings.Contains(cleanPath, "..") {
			return fmt.Errorf("discovery path contains path traversal: %s", path)
		}
		
		// Check for dangerous characters
		dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\"", "'"}
		for _, char := range dangerousChars {
			if strings.Contains(path, char) {
				return fmt.Errorf("discovery path contains dangerous character %s: %s", char, path)
			}
		}
	}
	
	// Validate plugin names (both enabled and disabled)
	allPluginNames := append(config.Enabled, config.Disabled...)
	for _, name := range allPluginNames {
		if name == "" {
			return fmt.Errorf("plugin name cannot be empty")
		}
		
		// Plugin names should be alphanumeric with dashes/underscores
		for _, char := range name {
			if !((char >= 'a' && char <= 'z') || 
				 (char >= 'A' && char <= 'Z') || 
				 (char >= '0' && char <= '9') || 
				 char == '-' || char == '_') {
				return fmt.Errorf("plugin name contains invalid character: %s", name)
			}
		}
	}
	
	// Check for conflicts between enabled and disabled
	enabledMap := make(map[string]bool)
	for _, name := range config.Enabled {
		enabledMap[name] = true
	}
	for _, name := range config.Disabled {
		if enabledMap[name] {
			return fmt.Errorf("plugin %s cannot be both enabled and disabled", name)
		}
	}
	
	return nil
}