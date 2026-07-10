from pathlib import Path

path = Path("internal/cli/app_test.go")
text = path.read_text()
text = text.replace(
    "func TestPreCommandAddressBecomesConnectAddressForRead(t *testing.T) {",
    "func TestPreCommandConnectAddressForRead(t *testing.T) {",
    1,
)
text = text.replace(
    'Run([]string{"--address", "192.0.2.10:502", "read",',
    'Run([]string{"--connect-address", "192.0.2.10:502", "read",',
    1,
)
path.write_text(text)
