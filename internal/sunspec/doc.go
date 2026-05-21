// Package sunspec is reserved for optional SunSpec-specific discovery and
// model access.
//
// Do not import github.com/andig/gosunspec/modbus here until its Modbus
// adapter is compatible with the context-aware github.com/grid-x/modbus API
// used by this CLI. Generic named points live in internal/devicemap and should
// remain independent from SunSpec auto-discovery.
package sunspec
