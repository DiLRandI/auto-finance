# Auto Finance

A Go application for automating financial operations with Google Sheets integration.

## Features

- Automated financial calculations
- Google Sheets integration
- Customizable financial rules
- Scheduled operations

## Installation

1. Clone the repository:
   ```bash
   git clone git@github-dilrandi:DiLRandI/auto-finance.git
   cd auto-finance
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the application:
   ```bash
   make build
   ```

## Configuration

Create a `.env` file in the project root with the following variables:
```
GOOGLE_CREDENTIALS_PATH=/path/to/credentials.json
SHEET_ID=your-google-sheet-id
```

## Usage

Run the application:
```bash
./bin/auto-finance
```

## Deployment

Deploy using the provided Kubernetes template:
```bash
kubectl apply -f deployment/template.yaml
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Submit a pull request

## License

MIT
