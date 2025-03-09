# Kafka Explorer CLI

A command-line utility for searching and reading messages from Apache Kafka topics with real-time display of results.

## Features

- Search for text using simple string matching or regular expressions
- Real-time TUI (Terminal User Interface) with progress and metrics
- Immediate saving of matching messages to JSON
- Parallel reading from multiple partitions for high performance
- Detailed metrics for monitoring search progress
- Configurable search options (case sensitivity, regex support)

## Installation

### From Source

1. Clone the repository:

   ```
   git clone https://github.com/Rejke/kafka-explorer-cli.git
   cd kafka-explorer-cli
   ```

2. Build the binary:

   ```
   go build -o kafka-explorer .
   ```

3. Install (optional):
   ```
   go install
   ```

### Using Go

```bash
go install github.com/Rejke/kafka-explorer-cli@latest
```

## Usage

### Basic Usage

```bash
kafka-explorer run --brokers=localhost:9092 --topic=my-topic --search="error" --output=results.json
```

### Command Line Options

#### Kafka Connection Parameters

- `--brokers`: Kafka broker addresses (comma-separated, default: "localhost:9092")
- `--topic`: Kafka topic to read from (required)
- `--group`: Consumer group ID (default: "kafka-explorer")

#### SASL Authentication Parameters

- `--sasl`: Enable SASL authentication (default: false)
- `--sasl-mechanism`: SASL mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512) (default: "PLAIN")
- `--security-protocol`: Security protocol (PLAINTEXT, SASL_PLAINTEXT, SSL, SASL_SSL) (default: "SASL_PLAINTEXT")
- `--username`: SASL username
- `--password`: SASL password

#### Search Parameters

- `--search`: String to search for in messages (required)
- `--limit`: Number of matches to find before stopping (0 = unlimited, default: 0)
- `--case-sensitive`: Enable case-sensitive search (default: false)
- `--regex`: Use regular expressions for search (default: false)

#### Output Parameters

- `--output`: Path to output JSON file (default: "results.json")
- `--verbose`: Enable verbose output (default: false)
- `--color`: Color mode: "auto", "always", or "never" (default: "auto")

#### Configuration

- `--config`: Config file (default is $HOME/.kafka-explorer.yaml)

### Example Commands

Search for "error" messages:

```bash
kafka-explorer run --brokers=localhost:9092 --topic=logs --search="error"
```

Search using a regular expression:

```bash
kafka-explorer run --brokers=localhost:9092 --topic=logs --search="error.\*timeout" --regex
```

Stop after finding 10 matches:

```bash
kafka-explorer run --brokers=localhost:9092 --topic=logs --search="error" --limit=10
```

Connect to Kafka with SASL authentication:

```bash
kafka-explorer run --brokers=kafka1:9092,kafka2:9092 --topic=logs --search="error" --sasl --sasl-mechanism="SCRAM-SHA-256" --security-protocol="SASL_PLAINTEXT" --username="user" --password="password"
```

## Output Format

The tool saves search results to a JSON file with the following format:

```json
{
  "brokers": ["kafka:9000", "kafka:9001", ...],
  "topic": "position-created",
  "searchString": "SEARCH",
  "limit": 0,
  "isCaseSensitive": false,
  "isRegexpSearch": false,
  "matches": [...]
}
```
