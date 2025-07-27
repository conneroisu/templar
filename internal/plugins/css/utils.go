package css

import (
	"strings"
)

// purgeUnusedClasses removes unused CSS classes from the provided CSS string.
// This is a common utility function used by multiple CSS framework plugins.
func purgeUnusedClasses(
	css string,
	usedClasses []string,
	shouldKeepRule func(string, map[string]bool) bool,
) string {
	// Create a map for fast lookup
	usedMap := make(map[string]bool)
	for _, class := range usedClasses {
		usedMap[class] = true
	}

	// Simple purging - remove class rules that aren't used
	// This is a basic implementation - a full implementation would use a CSS parser

	lines := strings.Split(css, "\n")
	var result []string

	inRule := false
	var currentRule strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, ".") && strings.Contains(line, "{"):
			// Start of a class rule
			inRule = true
			currentRule.Reset()
			currentRule.WriteString(line)
		case inRule:
			currentRule.WriteString("\n" + line)
			if strings.Contains(line, "}") {
				// End of rule
				rule := currentRule.String()
				if shouldKeepRule(rule, usedMap) {
					result = append(result, rule)
				}
				inRule = false
			}
		case !inRule:
			// Not in a class rule, keep as-is
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
