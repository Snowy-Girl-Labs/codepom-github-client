# CLAUDE.md

## Build and Run Commands
* **Go Gateway**:
  * Build: `go build -o bin/server ./gateway/cmd/server`
  * Run locally: `PORT=8000 go run ./gateway/cmd/server/main.go`
* **Python Worker**:
  * Install dependencies: `pip install -r py-agent/requirements.txt`
  * Run locally: `python py-agent/main.py`

## Code Style & Guidelines
* **Go**: Standard Go styling (`gofmt`). Standard directory structure.
* **Python**: PEP 8 styling. Strict Pydantic v2 schemas for contract validation between Go and Python.
* Keep client implementations lightweight and stateless. All complex model logic, prompt engineering, and key custody are delegated to the centralized CodePom Core service.
