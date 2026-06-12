# Contributing to tkngate

First off, thank you for considering contributing to tkngate! It's people like you that make open-source software such a great community.

## Development Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/tkngate/tkngate.git
   cd tkngate
   ```

2. **Install Go:**
   Ensure you have Go 1.20 or later installed.

3. **Install Dependencies:**
   ```bash
   go mod tidy
   ```

4. **Build the Proxy:**
   ```bash
   go build
   ```

## Pull Request Process

1. Ensure any install or build dependencies are removed before the end of the layer when doing a build.
2. Update the README.md with details of changes to the interface, this includes new environment variables, exposed ports, useful file locations and container parameters.
3. Increase the version numbers in any examples files and the README.md to the new version that this Pull Request would represent.
4. You may merge the Pull Request in once you have the sign-off of two other developers, or if you do not have permission to do that, you may request the second reviewer to merge it for you.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please be welcoming, respectful, and collaborative.
