# Loading message
echo "Loading Clutter project aliases..."

# Load aws credentials into env vars
source ./.vscode/.aws_credentials

# Directory aliases
alias cdapi="cd backend/api"
alias cdinfra="cd backend/infra"
alias cdinfradev="cd backend/infra-dev"

# Build Go binaries
alias gob="( cdapi && ./build.sh )"
alias golb="( cdapi && ./build_linux.sh )"

# DEVELOPMENT

# Terraform commands
alias tfi="( cdinfradev && terraform init )"
alias tfa="( cdinfradev && terraform apply )"
alias tfaa="( cdinfradev && terraform apply -auto-approve)"
alias tfia="tfi && tfa"
alias tfiaa="tfi && tfaa"

# Build go code and Terraform commands
alias tfbia="gob && tfia"   # Build, init and apply
alias tfbiaa="gob && tfiaa" # Build, init and apply auto-approve
alias tfba="gob && tfa"     # Build and apply
alias tfbaa="gob && tfaa"   # Build and apply auto-approve

# PRODUCTION

# Terraform commands
alias tfpi="( cdinfra && terraform init )"
alias tfpa="( cdinfra && terraform apply )"
alias tfpaa="( cdinfra && terraform apply -auto-approve)"
alias tfpia="tfpi && tfpa"
alias tfpiaa="tfpi && tfpaa"

# Build go code and Terraform commands
alias tfpbia="gob && tfpia"   # Build, init and apply
alias tfpbiaa="gob && tfpiaa" # Build, init and apply auto-approve
alias tfpba="gob && tfpa"     # Build and apply
alias tfpbaa="gob && tfpaa"   # Build and apply auto-approve

# Build go code (linux) and Terraform commands
alias tflbia="golb && tfia"   # Build, init and apply
alias tflbiaa="golb && tfiaa" # Build, init and apply auto-approve
alias tflba="golb && tfa"     # Build and apply
alias tflbaa="golb && tfaa"   # Build and apply auto-approve
