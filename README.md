# CodePom GitHub Client

This repository contains the **CodePom GitHub Client**, composed of:
1. A high-performance **Go Webhook Gateway & Queue** (`gateway/`) that enqueues incoming GitHub webhooks.
2. A **Python Worker** (`py-agent/`) that interacts with **CodePom Core** APIs to perform reviews/fixes and execute them in sandbox environments.

## Directory Structure
* `/gateway`: Go HTTP server & PostgreSQL Queue processor.
* `/py-agent`: Python Pydantic validation and agent execution worker.

## Getting Started
Refer to the individual `/gateway` and `/py-agent` directories for setup instructions.
