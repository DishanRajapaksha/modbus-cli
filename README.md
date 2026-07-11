# Modbus CLI

`modbus-cli` is a script-friendly command-line client for Modbus TCP and RTU.

## At a Glance

| Task | Command |
|---|---|
| Create starter config | `modbus-cli init-config` |
| Validate local config | `modbus-cli validate-config` |
| Test connectivity | `modbus-cli test-connection` |
| Read coils | `modbus-cli read coils --address 0 --quantity 8` |
| Read discrete inputs | `modbus-cli read discrete-inputs --address 0 --quantity 8` |
| Read holding registers | `modbus-cli read holding-registers --address 0 --quantity 2 --type float32` |
| Read input registers | `modbus-cli read input-registers --address 0 --quantity 2 --type uint16` |
| List named points | `modbus-cli points` |
| Read a named point | `modbus-cli read-point active_power` |
| Dry-run a named point write | `modbus-cli write-point breaker_closed --value on` |
| Dry-run a coil write | `modbus-cli write coil --address 0 --value on` |
| Execute a register write | `modbus-cli write register --address 10 --type uint16 --value 42 --yes` |
| Poll values | `modbus-cli watch holding-registers --address 0 --quantity 2 --interval 1s --format jsonl` |
| Read device identification | `modbus-cli identify --level basic` |
| Scan SunSpec models | `modbus-cli sunspec scan` |

## Install

Build from source:

```bash
make test
make build
```

Binary output: `bin/modbus-cli`

## First Run

Create `config.yaml`:

```bash
modbus-cli init-config
```

Validate it locally:

```bash
modbus-cli validate-config
```

Verify transport and a basic Modbus request:

```bash
modbus-cli test-connection
```

Read one value:

```bash
modbus-cli read holding-registers --address 0 --quantity 1 --type uint16
```

## Config and Profiles

Use a custom config file:

```bash
modbus-cli read coils --config site-a.yaml --address 0 --quantity 8
```

Use a profile from config:

```bash
modbus-cli read holding-registers --config config.yaml --profile serial --address 0 --quantity 2
```

Override config values with CLI flags:

```bash
modbus-cli read coils --transport tcp --connect-address 192.0.2.10:502 --unit-id 7 --address 0 --quantity 8
modbus-cli --connect-address 192.0.2.10:502 read coils --address 0 --quantity 8
```

Example config:

```yaml
connection:
  transport: tcp
  address: 127.0.0.1:502
  unit_id: 1
  timeout: 5s
  idle_timeout: 1m

rtu:
  baud_rate: 19200
  data_bits: 8
  stop_bits: 1
  parity: E

output:
  format: table

points:
  - name: active_power
    kind: holding-register
    address: 0
    quantity: 2
    type: float32
    byte_order: big
    word_order: high-low
    unit: kW
    scale: 1
    offset: 0
  - name: breaker_closed
    kind: coil
    address: 0
    quantity: 1
    writable: true

default_profile: local
profiles:
  local:
    connection:
      transport: tcp
      address: 127.0.0.1:502
      unit_id: 1
  serial:
    connection:
      transport: rtu
      address: /dev/ttyUSB0
      unit_id: 1
```

## Core Commands

### Status and Diagnostics

```bash
modbus-cli validate-config --config config.example.yaml
modbus-cli test-connection --transport tcp --connect-address 127.0.0.1:502 --unit-id 1
modbus-cli status --profile local
```

`status` is an alias for `test-connection`.

### Read Values

```bash
modbus-cli read coils --address 0 --quantity 8
modbus-cli read discrete-inputs --address 0 --quantity 8
modbus-cli read holding-registers --address 0 --quantity 2 --type float32
modbus-cli read input-registers --address 0 --quantity 4 --type uint16 --format json
modbus-cli read coils --connect-address 192.0.2.10:502 --unit-id 7 --address 0 --quantity 8
```

### Named Points

Named points let operators use configured names instead of raw addresses:

```bash
modbus-cli points
modbus-cli read-point active_power
modbus-cli write-point breaker_closed --value on
modbus-cli write-point breaker_closed --value on --yes
modbus-cli watch-point active_power --interval 1s --duration 30s --format jsonl
```

Point fields:

- `name`: unique point name.
- `kind`: `coil`, `discrete-input`, `holding-register`, or `input-register`.
- `address`: starting Modbus address.
- `quantity`: coil/register count.
- `type`: register decode type, such as `uint16` or `float32`.
- `byte_order` and `word_order`: register byte/word order.
- `unit`: display unit.
- `scale` and `offset`: applied after reads and reversed before writes.
- `writable`: required for `write-point`.

Register decode types:

- `raw`
- `uint16`, `int16`
- `uint32`, `int32`
- `uint64`, `int64`
- `float32`, `float64`

Multi-register values support:

```bash
modbus-cli read holding-registers --address 0 --quantity 2 --type float32 --byte-order big --word-order low-high
```

### Write Values

Writes are dry-run by default. Use `--yes` to send a real write request.

```bash
modbus-cli write coil --address 0 --value on
modbus-cli write coil --address 0 --value on --yes
modbus-cli write coils --address 0 --value on,off,true,false --yes
modbus-cli write register --address 10 --type uint16 --value 42 --yes
modbus-cli write registers --address 20 --type float32 --value 12.5 --yes
```

Raw register writes use hex bytes:

```bash
modbus-cli write registers --address 20 --type raw --value 3f800000 --yes
```

### Poll Values

```bash
modbus-cli watch coils --address 0 --quantity 8 --interval 1s
modbus-cli watch holding-registers --address 0 --quantity 2 --type float32 --duration 30s --format jsonl
```

`watch` is polling-based.

### Device Identification

```bash
modbus-cli identify --level basic
modbus-cli identify --level regular --format json
```

Supported levels are `basic`, `regular`, and `extended`.

## Write Safety

- Write commands print the target address, quantity, values, and send state.
- Write commands do not send by default.
- `--yes` sends the request.
- `--dry-run` and `--yes` cannot be used together.

## Output Formats

Snapshot commands support:

```text
table
text
json
csv
```

Stream commands support:

```text
text
jsonl
csv
```

Stream commands reject `--format json` because a stream is not one complete JSON document.

## Troubleshooting and Diagnostics

Use `validate-config` first to catch YAML and profile problems:

```bash
modbus-cli validate-config --config config.yaml --profile site-a
```

Use `test-connection --verbose` to print high-level connection decisions to stderr:

```bash
modbus-cli test-connection --verbose
```

Use `--debug` to enable lower-level Modbus frame logging where supported by the adapter:

```bash
modbus-cli read holding-registers --address 0 --quantity 1 --debug
```

## SunSpec

SunSpec support is separate from generic named points. The CLI uses the `github.com/andig/gosunspec` domain model and layout scanner, but it does not import `gosunspec/modbus`; instead it bridges SunSpec reads through this CLI's context-aware Modbus wrapper.

```bash
modbus-cli sunspec scan
modbus-cli sunspec models --format json
modbus-cli sunspec read --model 1 --point Mn
```

`sunspec scan` and `sunspec models` discover the SunSpec marker and list detected models, blocks, addresses, lengths, and point counts. `sunspec read` reads one point from a detected model.

## Shell Completions

```bash
# bash
modbus-cli completions bash > /etc/bash_completion.d/modbus-cli

# zsh
mkdir -p "${HOME}/.zsh/completions"
modbus-cli completions zsh > "${HOME}/.zsh/completions/_modbus-cli"
```

## Exit Codes

```text
0  success
1  general error
2  config or validation error
3  connection error
4  protocol or request error
7  write or control rejected (reserved)
8  operation timeout
9  output or formatting error
```

## Command Help

```bash
modbus-cli help
modbus-cli read --help
modbus-cli write --help
modbus-cli watch --help
```
