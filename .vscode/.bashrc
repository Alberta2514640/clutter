# Loading message
echo "Loading Clutter project aliases..."

# Load aws credentials into env vars
source ./.vscode/.aws_credentials

# Directory aliases
alias cdapi="cd backend/api"
alias cdinfra="cd backend/infra"

# Build Go binaries
alias gob="( cdapi && ./build.sh )"
alias golb="( cdapi && ./build_linux.sh )"

# Terraform commands
alias tfi="( cdinfra && terraform init )"
alias tfa="( cdinfra && terraform apply )"
alias tfaa="( cdinfra && terraform apply -auto-approve)"
alias tfia="tfi && tfa"
alias tfiaa="tfi && tfaa"

# Build go code and Terraform commands
alias tfbia="gob && tfia"   # Build, init and apply
alias tfbiaa="gob && tfiaa" # Build, init and apply auto-approve
alias tfba="gob && tfa"           # Build and apply
alias tfbaa="gob && tfaa"         # Build and apply auto-approve

# Build go code (linux) and Terraform commands
alias tflbia="golb && tfia"   # Build, init and apply
alias tflbiaa="golb && tfiaa" # Build, init and apply auto-approve
alias tflba="golb && tfa"           # Build and apply
alias tflbaa="golb && tfaa"         # Build and apply auto-approve