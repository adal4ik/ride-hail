
#!/bin/bash

# Check for required arguments
if [ -z "$1" ] || [ -z "$2" ]; then
  echo "Usage: $0 <name> <email>"
  exit 1
fi

# Set Git config
git config --global user.name "$1"
git config --global user.email "$2"

# Confirm what was set
echo "Git user.name set to: $(git config --global user.name)"
echo "Git user.email set to: $(git config --global user.email)"
