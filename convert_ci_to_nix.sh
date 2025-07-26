#!/bin/bash

# Script to convert CI workflow from Go setup to Nix

FILE="/home/connerohnesorge/Documents/001Repos/templar/.github/workflows/ci.yml"

# Create a backup
cp "$FILE" "$FILE.backup"

# Replace Go setup with Nix setup
sed -i '
# Remove Go version strategy from unit tests section
/strategy:/,/matrix:/ {
  /go-version:/d
}

# Replace Set up Go with Install Nix
s/- name: Set up Go/- name: Install Nix/g
s/uses: actions\/setup-go@v5/uses: DeterminateSystems\/nix-installer-action@main/g

# Remove with: go-version lines
/with:/,/go-version:/ {
  /with:/d
  /go-version:/d
}

# Replace Cache Go modules sections
/- name: Cache Go modules/,/restore-keys:/ {
  /- name: Cache Go modules/c\
      - name: Setup Nix cache\
        uses: DeterminateSystems/magic-nix-cache-action@main
  /uses: actions\/cache@v4/d
  /with:/d
  /path:/d
  /~\/\.cache\/go-build/d
  /~\/go\/pkg\/mod/d
  /key:/d
  /restore-keys:/d
}

# Replace Install dependencies + templ sections
/- name: Install dependencies/,/go generate/ {
  /- name: Install dependencies/c\
      - name: Setup development environment\
        run: nix develop --command bash -c "go mod download && templ generate && go generate ./..."
  /run: go mod download/d
  /- name: Generate Go code/d
  /run: |/d
  /go install.*templ/d
  /go generate/d
}

# Replace standalone go commands with nix develop commands
s/run: go test/run: nix develop --command go test/g
s/run: go build/run: nix develop --command go build/g
s/run: go vet/run: nix develop --command go vet/g

' "$FILE"

echo "Converted CI workflow to use Nix"
echo "Backup saved as $FILE.backup"