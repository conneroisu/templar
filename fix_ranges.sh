#!/bin/bash

# Simple shell script to fix range violations in cmd/audit.go

file="cmd/audit.go"

# Fix for _, violation := range errorViolations
sed -i 's/for _, violation := range errorViolations {/for i := range errorViolations {\n\t\tviolation := \&errorViolations[i]/g' "$file"

# Fix for _, violation := range warningViolations  
sed -i 's/for _, violation := range warningViolations {/for i := range warningViolations {\n\t\tviolation := \&warningViolations[i]/g' "$file"

# Fix for _, violation := range infoViolations
sed -i 's/for _, violation := range infoViolations {/for i := range infoViolations {\n\t\tviolation := \&infoViolations[i]/g' "$file"

# Fix for _, violation := range report.Violations (any remaining)
sed -i 's/for _, violation := range report\.Violations {/for i := range report.Violations {\n\t\tviolation := \&report.Violations[i]/g' "$file"

# Fix for _, violation := range violations
sed -i 's/for _, violation := range violations {/for i := range violations {\n\t\tviolation := \&violations[i]/g' "$file"

echo "Applied range fixes to $file"