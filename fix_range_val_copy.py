#!/usr/bin/env python3

import re
import sys

def fix_range_val_copy(content):
    """Fix rangeValCopy violations by converting range over values to range over indices."""
    
    # Pattern 1: for _, violation := range report.Violations
    pattern1 = r'for _, violation := range (\w+)\.Violations \{'
    replacement1 = r'for i := range \1.Violations {\n\t\tviolation := &\1.Violations[i]'
    content = re.sub(pattern1, replacement1, content)
    
    # Pattern 2: for _, violation := range errorViolations (and similar local slices)
    pattern2 = r'for _, violation := range (\w+Violations) \{'
    replacement2 = r'for i := range \1 {\n\t\tviolation := &\1[i]'
    content = re.sub(pattern2, replacement2, content)
    
    # Pattern 3: for _, suggestion := range violation.Suggestions
    pattern3 = r'for _, suggestion := range violation\.Suggestions \{'
    replacement3 = r'for i := range violation.Suggestions {\n\t\t\t\tsuggestion := &violation.Suggestions[i]'
    content = re.sub(pattern3, replacement3, content)
    
    # Pattern 4: for _, item := range quickStart
    pattern4 = r'for i, item := range (\w+) \{'
    replacement4 = r'for i := range \1 {\n\t\titem := &\1[i]'
    content = re.sub(pattern4, replacement4, content)
    
    # Pattern 5: for _, violation := range violations (generic)
    pattern5 = r'for _, violation := range violations \{'
    replacement5 = r'for i := range violations {\n\t\tviolation := &violations[i]'
    content = re.sub(pattern5, replacement5, content)
    
    # Pattern 6: for _, suggestion := range suggestions
    pattern6 = r'for _, suggestion := range (\w+) \{'
    replacement6 = r'for i := range \1 {\n\t\tsuggestion := &\1[i]'
    content = re.sub(pattern6, replacement6, content)
    
    # Now we need to update any references to these variables to use pointer dereferencing
    # For appends, change from "item" to "*item"
    content = re.sub(r'append\((\w+), violation\)', r'append(\1, *violation)', content)
    content = re.sub(r'append\((\w+), suggestion\)', r'append(\1, *suggestion)', content)
    content = re.sub(r'append\((\w+), item\)', r'append(\1, *item)', content)
    
    return content

# Read the file
if len(sys.argv) != 2:
    print("Usage: python3 fix_range_val_copy.py <file>")
    sys.exit(1)

filename = sys.argv[1]
try:
    with open(filename, 'r') as f:
        content = f.read()
    
    # Apply fixes
    fixed_content = fix_range_val_copy(content)
    
    # Write back
    with open(filename, 'w') as f:
        f.write(fixed_content)
    
    print(f"Fixed rangeValCopy violations in {filename}")
    
except Exception as e:
    print(f"Error: {e}")
    sys.exit(1)