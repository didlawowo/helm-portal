#!/bin/bash

# 🎯 Setup script for helm-portal project

# ❌ Error handling
set -e
trap 'echo "❌ Error on line $LINENO. Exit code: $?"' ERR

# 📂 Create main project directory
echo "🚀 Creating project structure..."
mkdir -p src/{cmd/server,internal/{api/{handlers,middleware,routes},chart/{parser,storage},kubernetes/client,models},web/{templates,static},config}

# 📝 Create initial files
cd src

# Create go.mod
echo "📦 Initializing Go module..."
go mod init helm-portal

# Create main.go
cat > cmd/server/main.go << 'EOF'
package main

import (
    "log"
    "github.com/gofiber/fiber/v2"
)

func main() {
    app := fiber.New()
    
    log.Fatal(app.Listen(":3000"))
}
EOF

# Create config file
cat > config/config.yaml << 'EOF'
server:
  port: 3000
  host: "localhost"

storage:
  path: "./charts"

auth:
  enabled: true
  
kubernetes:
  configPath: "~/.kube/config"
EOF

# Create .gitignore
cat > .gitignore << 'EOF'
# Binaries and objects
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out

# IDE specific files
.idea/
.vscode/
*.swp
*.swo

# OS specific files
.DS_Store
.env

# Project specific
/charts
/tmp
EOF

# 🎨 Create basic HTML template
mkdir -p web/templates/layouts
cat > web/templates/layouts/main.html << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Helm Portal</title>
    <link rel="stylesheet" href="/static/css/main.css">
</head>
<body>
    <nav>
        <h1>Helm Portal</h1>
    </nav>
    <main>
        {{embed}}
    </main>
    <script src="/static/js/main.js"></script>
</body>
</html>
EOF

# Create basic CSS
mkdir -p web/static/css
cat > web/static/css/main.css << 'EOF'
body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    margin: 0;
    padding: 0;
    background: #f5f5f5;
}

nav {
    background: #2d3748;
    color: white;
    padding: 1rem;
}

main {
    padding: 2rem;
}
EOF

# Create basic JS
mkdir -p web/static/js
cat > web/static/js/main.js << 'EOF'
// Main JavaScript file
console.log('Helm Portal initialized');
EOF

# 📦 Install required Go packages
echo "📦 Installing required Go packages..."
go get github.com/gofiber/fiber/v2
go get helm.sh/helm/v3@latest
go get k8s.io/client-go@latest

echo "✅ Project structure created successfully!"
echo "📁 Project initialized at: $(pwd)"
echo "🚀 Run 'cd helm-portal && go run cmd/server/main.go' to start the server"