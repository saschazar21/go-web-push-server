set -euxo pipefail

# Create the functions directory
mkdir -p "$(pwd)/functions"

# Build the functions
GOBIN="$(pwd)/functions" go install ./...
chmod +x "$(pwd)/functions/*"

# Print the environment variables
go env