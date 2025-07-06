#!/bin/bash
# migrate.sh - Create Docker secrets from .env file
# Unix/Linux equivalent of migrate.ps1

set -euo pipefail  # Exit on error, undefined vars, and pipe failures

# --- Configuration ---
ENV_FILE=".env"
SECRETS_DIR="./secrets"
# --- End Configuration ---

# Check if .env file exists
if [[ ! -f "$ENV_FILE" ]]; then
    echo "Error: The '$ENV_FILE' file was not found in the current directory." >&2
    exit 1
fi

# Create secrets directory if it doesn't exist
if [[ ! -d "$SECRETS_DIR" ]]; then
    echo "Creating secrets directory at '$SECRETS_DIR'..."
    mkdir -p "$SECRETS_DIR"
fi

# Declare associative arrays for environment variables
declare -A env_vars
declare -A resolved_vars

echo "Reading environment variables from '$ENV_FILE'..."

# --- Pass 1: Read all variables ---
while IFS= read -r line || [[ -n "$line" ]]; do
    # Trim whitespace
    line=$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    
    # Skip empty lines and comments
    if [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]]; then
        continue
    fi
    
    # Check if line contains '='
    if [[ "$line" =~ ^([^=]+)=(.*)$ ]]; then
        key="${BASH_REMATCH[1]}"
        value="${BASH_REMATCH[2]}"
        
        # Trim whitespace from key and value
        key=$(echo "$key" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        value=$(echo "$value" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        
        # Remove surrounding quotes if present
        if [[ "$value" =~ ^[\"\'](.*)[\"\']$ ]]; then
            value="${BASH_REMATCH[1]}"
        fi
        
        env_vars["$key"]="$value"
    fi
done < "$ENV_FILE"

# --- Pass 2: Resolve nested variables ---
echo "Resolving variable references..."

for key in "${!env_vars[@]}"; do
    value="${env_vars[$key]}"
    original_value="$value"
    
    # Keep resolving until no more substitutions are made
    while true; do
        new_value="$value"
        
        # Find and replace ${VAR} and $VAR patterns
        while [[ "$new_value" =~ \$\{?([A-Za-z_][A-Za-z0-9_]*)\}? ]]; do
            var_name="${BASH_REMATCH[1]}"
            var_pattern="${BASH_REMATCH[0]}"
            
            if [[ -n "${env_vars[$var_name]:-}" ]]; then
                replacement="${env_vars[$var_name]}"
                new_value="${new_value/$var_pattern/$replacement}"
            else
                # Variable not found, leave as is and break to avoid infinite loop
                break
            fi
        done
        
        # If no changes were made, we're done
        if [[ "$new_value" == "$value" ]]; then
            break
        fi
        
        value="$new_value"
    done
    
    resolved_vars["$key"]="$value"
    
    # Show what was resolved (optional debug info)
    if [[ "$original_value" != "$value" ]]; then
        echo "  Resolved $key: '$original_value' → '$value'"
    fi
done

# --- Pass 3: Write the resolved variables to secret files ---
echo "Writing secrets to files..."

for key in "${!resolved_vars[@]}"; do
    # Convert key to lowercase for filename
    secret_filename=$(echo "$key" | tr '[:upper:]' '[:lower:]').txt
    secret_filepath="$SECRETS_DIR/$secret_filename"
    
    # Write the value to the file (UTF-8, no BOM)
    printf '%s' "${resolved_vars[$key]}" > "$secret_filepath"
    
    echo "  - Created '$secret_filepath'"
done

echo "Secret files created successfully."
echo ""
echo "Next steps:"
echo "  1. Review the generated files in '$SECRETS_DIR/'"
echo "  2. Run: docker-compose up -d"