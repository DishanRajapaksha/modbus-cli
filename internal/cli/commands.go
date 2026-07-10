package cli

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DishanRajapaksha/modbus-cli/internal/config"
	"github.com/DishanRajapaksha/modbus-cli/internal/devicemap"
	"github.com/DishanRajapaksha/modbus-cli/internal/modbusclient"
	"github.com/DishanRajapaksha/modbus-cli/internal/output"
	"github.com/DishanRajapaksha/modbus-cli/internal/sunspec"
)

func (a *App) initConfig(args []string) error {
	fs := a.newFlagSet("init-config")
	outputPath := config.DefaultConfigPath
	force := false
	fs.StringVar(&outputPath, "output", outputPath, "output YAML config file")
	fs.BoolVar(&force, "force", false, "overwrite output file if it exists")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !force {
		if _, err := os.Stat(outputPath); err == nil {
			return fmt.Errorf("%w: refusing to overwrite existing file %q; use --force to overwrite", config.ErrConfig, outputPath)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: stat %q: %v", config.ErrConfig, outputPath, err)
		}
	}
	contents, err := config.StarterConfigYAML()
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, contents, 0o600); err != nil {
		return fmt.Errorf("%w: write config %q: %v", config.ErrConfig, outputPath, err)
	}
	fmt.Fprintf(a.out, "wrote starter config to %s\n", outputPath)
	return nil
}

func (a *App) validateConfig(args []string) error {
	fs := a.newFlagSet("validate-config")
	common := commonOptions{}
	addCommonFlags(fs, &common, output.FormatTable, "output format", true, true)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if _, err := os.Stat(common.configPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: config file %q not found; run modbus-cli init-config", config.ErrConfig, common.configPath)
		}
		return fmt.Errorf("%w: stat config %q: %v", config.ErrConfig, common.configPath, err)
	}
	if _, _, err := common.loadConfig(fs, output.FormatTable); err != nil {
		return err
	}
	fmt.Fprintln(a.out, "config validation: PASS")
	return nil
}

func (a *App) testConnection(args []string) error {
	fs := a.newFlagSet("test-connection")
	common := commonOptions{}
	addCommonFlags(fs, &common, output.FormatTable, "output format: table, text, json, or csv", true, true)
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, format, err := common.loadConfig(fs, output.FormatTable)
	if err != nil {
		return err
	}
	if err := validateSnapshotFormat(format); err != nil {
		return err
	}
	result := modbusclient.DiagnosticResult{
		Transport: cfg.Connection.Transport,
		Address:   cfg.Connection.Address,
		UnitID:    cfg.Connection.UnitID,
		Result:    "PASS",
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout)
	defer cancel()
	client, err := a.factory.New(cfg, modbusclient.Options{Verbose: common.verbose, Debug: common.debug})
	if err != nil {
		result.Result = "FAIL"
		result.Error = err.Error()
		_ = a.renderDiagnostic(format, result)
		return err
	}
	if err := client.Connect(ctx); err != nil {
		result.Result = "FAIL"
		result.Error = err.Error()
		_ = a.renderDiagnostic(format, result)
		return err
	}
	defer client.Close()
	if _, err := client.ReadCoils(ctx, 0, 1); err != nil {
		result.Result = "FAIL"
		result.Error = err.Error()
		_ = a.renderDiagnostic(format, result)
		return err
	}
	return a.renderDiagnostic(format, result)
}

func (a *App) read(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: read type is required", modbusclient.ErrValidation)
	}
	if args[0] == "--help" || args[0] == "-h" {
		printReadHelp(a.err)
		return flag.ErrHelp
	}
	kind := args[0]
	fs := a.newFlagSet("read " + kind)
	common := commonOptions{}
	addCommonFlags(fs, &common, output.FormatTable, "output format: table, text, json, or csv", true, false)
	address := fs.Uint("address", 0, "starting Modbus address")
	quantity := fs.Uint("quantity", 1, "number of coils/registers to read")
	valueType := fs.String("type", modbusclient.TypeRaw, "register type: raw, uint16, int16, uint32, int32, uint64, int64, float32, or float64")
	byteOrder := fs.String("byte-order", "big", "register byte order: big or little")
	wordOrder := fs.String("word-order", "high-low", "multi-register word order: high-low or low-high")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	cfg, format, err := common.loadConfig(fs, output.FormatTable)
	if err != nil {
		return err
	}
	if err := validateSnapshotFormat(format); err != nil {
		return err
	}
	if *address > 65535 || *quantity == 0 || *quantity > 2000 {
		return fmt.Errorf("%w: invalid --address or --quantity", modbusclient.ErrValidation)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout)
	defer cancel()
	client, err := a.openClient(ctx, cfg, common)
	if err != nil {
		return err
	}
	defer client.Close()
	result, err := readOnce(ctx, client, kind, uint16(*address), uint16(*quantity), *valueType, *byteOrder, *wordOrder)
	if err != nil {
		return err
	}
	return a.renderRead(format, result)
}

func readOnce(ctx context.Context, client modbusclient.Client, kind string, address, quantity uint16, valueType, byteOrder, wordOrder string) (modbusclient.ReadResult, error) {
	result := modbusclient.ReadResult{Kind: kind, Address: address, Quantity: quantity, Type: valueType, Timestamp: time.Now()}
	switch kind {
	case "coils":
		data, err := client.ReadCoils(ctx, address, quantity)
		if err != nil {
			return result, err
		}
		values := modbusclient.DecodeCoils(data, quantity)
		for i, value := range values {
			result.Values = append(result.Values, modbusclient.Value{Address: address + uint16(i), Value: value})
		}
	case "discrete-inputs":
		data, err := client.ReadDiscreteInputs(ctx, address, quantity)
		if err != nil {
			return result, err
		}
		values := modbusclient.DecodeCoils(data, quantity)
		for i, value := range values {
			result.Values = append(result.Values, modbusclient.Value{Address: address + uint16(i), Value: value})
		}
	case "holding-registers", "input-registers":
		var data []byte
		var err error
		if kind == "holding-registers" {
			data, err = client.ReadHoldingRegisters(ctx, address, quantity)
		} else {
			data, err = client.ReadInputRegisters(ctx, address, quantity)
		}
		if err != nil {
			return result, err
		}
		result.Raw = strings.ToUpper(hex.EncodeToString(data))
		values, err := modbusclient.DecodeRegisters(data, valueType, byteOrder, wordOrder)
		if err != nil {
			return result, err
		}
		for i := range values {
			values[i].Address += address
		}
		result.Values = values
	default:
		return result, fmt.Errorf("%w: unsupported read type %q", modbusclient.ErrValidation, kind)
	}
	return result, nil
}

func (a *App) write(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: write type is required", modbusclient.ErrValidation)
	}
	if args[0] == "--help" || args[0] == "-h" {
		printWriteHelp(a.err)
		return flag.ErrHelp
	}
	kind := args[0]
	fs := a.newFlagSet("write " + kind)
	common := commonOptions{}
	addCommonFlags(fs, &common, output.FormatTable, "output format: table, text, json, or csv", true, false)
	address := fs.Uint("address", 0, "starting Modbus address")
	rawValue := fs.String("value", "", "value to write; use comma-separated values for multiple coils/registers")
	valueType := fs.String("type", modbusclient.TypeUint16, "register type: raw, uint16, int16, uint32, int32, uint64, int64, float32, or float64")
	byteOrder := fs.String("byte-order", "big", "register byte order: big or little")
	wordOrder := fs.String("word-order", "high-low", "multi-register word order: high-low or low-high")
	dryRun := fs.Bool("dry-run", false, "print request without sending")
	yes := fs.Bool("yes", false, "send the write request")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *dryRun && *yes {
		return fmt.Errorf("%w: --dry-run and --yes cannot be used together", modbusclient.ErrValidation)
	}
	cfg, format, err := common.loadConfig(fs, output.FormatTable)
	if err != nil {
		return err
	}
	if err := validateSnapshotFormat(format); err != nil {
		return err
	}
	if *address > 65535 || strings.TrimSpace(*rawValue) == "" {
		return fmt.Errorf("%w: --address and --value are required", modbusclient.ErrValidation)
	}
	result, coilValues, registers, err := buildWriteResult(kind, uint16(*address), *rawValue, *valueType, *byteOrder, *wordOrder, !*yes)
	if err != nil {
		return err
	}
	if !*yes {
		return a.renderWrite(format, result)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout)
	defer cancel()
	client, err := a.openClient(ctx, cfg, common)
	if err != nil {
		return err
	}
	defer client.Close()
	switch kind {
	case "coil":
		if len(coilValues) != 1 {
			return fmt.Errorf("%w: write coil expects exactly one value", modbusclient.ErrValidation)
		}
		_, err = client.WriteSingleCoil(ctx, uint16(*address), coilValues[0])
	case "coils":
		_, err = client.WriteMultipleCoils(ctx, uint16(*address), coilValues)
	case "register":
		if len(registers) != 1 {
			return fmt.Errorf("%w: write register expects exactly one 16-bit register value", modbusclient.ErrValidation)
		}
		_, err = client.WriteSingleRegister(ctx, uint16(*address), registers[0])
	case "registers":
		_, err = client.WriteMultipleRegisters(ctx, uint16(*address), registers)
	default:
		return fmt.Errorf("%w: unsupported write type %q", modbusclient.ErrValidation, kind)
	}
	if err != nil {
		return err
	}
	result.DryRun = false
	result.Sent = true
	return a.renderWrite(format, result)
}

func buildWriteResult(kind string, address uint16, rawValue, valueType, byteOrder, wordOrder string, dryRun bool) (modbusclient.WriteResult, []bool, []uint16, error) {
	result := modbusclient.WriteResult{Kind: kind, Address: address, Type: valueType, DryRun: dryRun, Timestamp: time.Now()}
	switch kind {
	case "coil", "coils":
		values, err := modbusclient.ParseCoilValues(rawValue)
		if err != nil {
			return result, nil, nil, err
		}
		if kind == "coil" && len(values) != 1 {
			return result, nil, nil, fmt.Errorf("%w: write coil expects exactly one value", modbusclient.ErrValidation)
		}
		result.Quantity = uint16(len(values))
		for _, value := range values {
			result.Values = append(result.Values, value)
		}
		return result, values, nil, nil
	case "register", "registers":
		registers, display, err := modbusclient.EncodeRegisterValues(splitCSV(rawValue), valueType, byteOrder, wordOrder)
		if err != nil {
			return result, nil, nil, err
		}
		if kind == "register" && len(registers) != 1 {
			return result, nil, nil, fmt.Errorf("%w: write register expects exactly one 16-bit register value", modbusclient.ErrValidation)
		}
		result.Quantity = uint16(len(registers))
		result.Values = display
		return result, nil, registers, nil
	default:
		return result, nil, nil, fmt.Errorf("%w: unsupported write type %q", modbusclient.ErrValidation, kind)
	}
}

func (a *App) identify(args []string) error {
	fs := a.newFlagSet("identify")
	common := commonOptions{}
	addCommonFlags(fs, &common, output.FormatTable, "output format: table, text, json, or csv", true, true)
	levelValue := fs.String("level", string(modbusclient.IdentificationBasic), "identification level: basic, regular, or extended")
	if err := fs.Parse(args); err != nil {
		return err
	}
	level, err := modbusclient.ParseIdentificationLevel(*levelValue)
	if err != nil {
		return err
	}
	cfg, format, err := common.loadConfig(fs, output.FormatTable)
	if err != nil {
		return err
	}
	if err := validateSnapshotFormat(format); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout)
	defer cancel()
	client, err := a.openClient(ctx, cfg, common)
	if err != nil {
		return err
	}
	defer client.Close()
	objects, err := client.ReadDeviceIdentification(ctx, level)
	if err != nil {
		return err
	}
	result := modbusclient.IdentificationResult{Level: string(level), Objects: map[string]string{}}
	for id, value := range objects {
		result.Objects[identifyObjectName(id)] = string(value)
	}
	return a.renderIdentification(format, result)
}

func (a *App) points(args []string) error {
	fs := a.newFlagSet("points")
	common := commonOptions{}
	addCommonFlags(fs, &common, output.FormatTable, "output format: table, text, json, or csv", true, true)
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, format, err := common.loadConfig(fs, output.FormatTable)
	if err != nil {
		return err
	}
	if err := validateSnapshotFormat(format); err != nil {
		return err
	}
	return a.renderPoints(format, devicemap.List(cfg.Points))
}

func (a *App) readPoint(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: point name is required", modbusclient.ErrValidation)
	}
	if args[0] == "--help" || args[0] == "-h" {
		printPointHelp(a.err)
		return flag.ErrHelp
	}
	fs := a.newFlagSet("read-point")
	common := commonOptions{}
	addCommonFlags(fs, &common, output.FormatTable, "output format: table, text, json, or csv", true, false)
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	cfg, format, err := common.loadConfig(fs, output.FormatTable)
	if err != nil {
		return err
	}
	if err := validateSnapshotFormat(format); err != nil {
		return err
	}
	point, err := devicemap.Find(cfg.Points, args[0])
	if err != nil {
		return err
	}
	kind, err := devicemap.ReadKind(point)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout)
	defer cancel()
	client, err := a.openClient(ctx, cfg, common)
	if err != nil {
		return err
	}
	defer client.Close()
	result, err := readOnce(ctx, client, kind, point.Address, point.Quantity, point.Type, point.ByteOrder, point.WordOrder)
	if err != nil {
		return err
	}
	return a.renderRead(format, devicemap.ApplyRead(point, result))
}

func (a *App) writePoint(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: point name is required", modbusclient.ErrValidation)
	}
	if args[0] == "--help" || args[0] == "-h" {
		printPointHelp(a.err)
		return flag.ErrHelp
	}
	fs := a.newFlagSet("write-point")
	common := commonOptions{}
	addCommonFlags(fs, &common, output.FormatTable, "output format: table, text, json, or csv", true, false)
	rawValue := fs.String("value", "", "value to write")
	dryRun := fs.Bool("dry-run", false, "print request without sending")
	yes := fs.Bool("yes", false, "send the write request")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *dryRun && *yes {
		return fmt.Errorf("%w: --dry-run and --yes cannot be used together", modbusclient.ErrValidation)
	}
	cfg, format, err := common.loadConfig(fs, output.FormatTable)
	if err != nil {
		return err
	}
	if err := validateSnapshotFormat(format); err != nil {
		return err
	}
	point, err := devicemap.Find(cfg.Points, args[0])
	if err != nil {
		return err
	}
	kind, err := devicemap.WriteKind(point)
	if err != nil {
		return err
	}
	preparedValue, err := devicemap.PrepareWriteValue(point, *rawValue)
	if err != nil {
		return err
	}
	result, coilValues, registers, err := buildWriteResult(kind, point.Address, preparedValue, point.Type, point.ByteOrder, point.WordOrder, !*yes)
	if err != nil {
		return err
	}
	if result.Quantity != point.Quantity {
		return fmt.Errorf("%w: point %q expects %d value register/coil(s), got %d", modbusclient.ErrValidation, point.Name, point.Quantity, result.Quantity)
	}
	result.Point = point.Name
	result.Unit = point.Unit
	result.Values = anyValues(splitCSV(*rawValue))
	if !*yes {
		return a.renderWrite(format, result)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout)
	defer cancel()
	client, err := a.openClient(ctx, cfg, common)
	if err != nil {
		return err
	}
	defer client.Close()
	switch kind {
	case "coil":
		_, err = client.WriteSingleCoil(ctx, point.Address, coilValues[0])
	case "coils":
		_, err = client.WriteMultipleCoils(ctx, point.Address, coilValues)
	case "register":
		_, err = client.WriteSingleRegister(ctx, point.Address, registers[0])
	case "registers":
		_, err = client.WriteMultipleRegisters(ctx, point.Address, registers)
	default:
		return fmt.Errorf("%w: unsupported write type %q", modbusclient.ErrValidation, kind)
	}
	if err != nil {
		return err
	}
	result.DryRun = false
	result.Sent = true
	return a.renderWrite(format, result)
}

func (a *App) watch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: watch type is required", modbusclient.ErrValidation)
	}
	if args[0] == "--help" || args[0] == "-h" {
		printWatchHelp(a.err)
		return flag.ErrHelp
	}
	kind := args[0]
	fs := a.newFlagSet("watch " + kind)
	common := commonOptions{}
	addCommonFlags(fs, &common, output.FormatText, defaultStreamHelp, true, false)
	address := fs.Uint("address", 0, "starting Modbus address")
	quantity := fs.Uint("quantity", 1, "number of coils/registers to read")
	valueType := fs.String("type", modbusclient.TypeRaw, "register type")
	byteOrder := fs.String("byte-order", "big", "register byte order: big or little")
	wordOrder := fs.String("word-order", "high-low", "multi-register word order: high-low or low-high")
	interval := fs.Duration("interval", time.Second, "poll interval")
	duration := fs.Duration("duration", 0, "stop after this duration; zero runs until interrupted")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	cfg, format, err := common.loadConfig(fs, output.FormatText)
	if err != nil {
		return err
	}
	if err := validateStreamFormat(format); err != nil {
		return err
	}
	if *interval <= 0 || *address > 65535 || *quantity == 0 {
		return fmt.Errorf("%w: invalid --address, --quantity, or --interval", modbusclient.ErrValidation)
	}
	connectCtx, connectCancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout)
	defer connectCancel()
	client, err := a.openClient(connectCtx, cfg, common)
	if err != nil {
		return err
	}
	defer client.Close()
	runCtx := context.Background()
	if *duration > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(runCtx, *duration)
		defer cancel()
	}
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		result, err := readOnce(runCtx, client, kind, uint16(*address), uint16(*quantity), *valueType, *byteOrder, *wordOrder)
		if err != nil {
			return err
		}
		if err := a.renderWatch(format, result); err != nil {
			return err
		}
		select {
		case <-runCtx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (a *App) watchPoint(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: point name is required", modbusclient.ErrValidation)
	}
	if args[0] == "--help" || args[0] == "-h" {
		printPointHelp(a.err)
		return flag.ErrHelp
	}
	fs := a.newFlagSet("watch-point")
	common := commonOptions{}
	addCommonFlags(fs, &common, output.FormatText, defaultStreamHelp, true, false)
	interval := fs.Duration("interval", time.Second, "poll interval")
	duration := fs.Duration("duration", 0, "stop after this duration; zero runs until interrupted")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	cfg, format, err := common.loadConfig(fs, output.FormatText)
	if err != nil {
		return err
	}
	if err := validateStreamFormat(format); err != nil {
		return err
	}
	if *interval <= 0 {
		return fmt.Errorf("%w: --interval must be greater than zero", modbusclient.ErrValidation)
	}
	point, err := devicemap.Find(cfg.Points, args[0])
	if err != nil {
		return err
	}
	kind, err := devicemap.ReadKind(point)
	if err != nil {
		return err
	}
	connectCtx, connectCancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout)
	defer connectCancel()
	client, err := a.openClient(connectCtx, cfg, common)
	if err != nil {
		return err
	}
	defer client.Close()
	runCtx := context.Background()
	if *duration > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(runCtx, *duration)
		defer cancel()
	}
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		result, err := readOnce(runCtx, client, kind, point.Address, point.Quantity, point.Type, point.ByteOrder, point.WordOrder)
		if err != nil {
			return err
		}
		if err := a.renderWatch(format, devicemap.ApplyRead(point, result)); err != nil {
			return err
		}
		select {
		case <-runCtx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (a *App) sunspec(args []string) error {
	if len(args) == 0 {
		printSunSpecHelp(a.err)
		return fmt.Errorf("%w: sunspec command is required", modbusclient.ErrValidation)
	}
	switch args[0] {
	case "--help", "-h":
		printSunSpecHelp(a.err)
		return flag.ErrHelp
	case "scan", "models":
		return a.sunspecScan(args[1:])
	case "read":
		return a.sunspecRead(args[1:])
	default:
		return fmt.Errorf("%w: unknown sunspec command %q", modbusclient.ErrValidation, args[0])
	}
}

func (a *App) sunspecScan(args []string) error {
	fs := a.newFlagSet("sunspec scan")
	common := commonOptions{}
	addCommonFlags(fs, &common, output.FormatTable, "output format: table, text, json, or csv", true, true)
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, format, err := common.loadConfig(fs, output.FormatTable)
	if err != nil {
		return err
	}
	if err := validateSnapshotFormat(format); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout)
	defer cancel()
	client, err := a.openClient(ctx, cfg, common)
	if err != nil {
		return err
	}
	defer client.Close()
	result, err := sunspec.NewScanner(client).Scan(ctx)
	if err != nil {
		return err
	}
	return a.renderSunSpecScan(format, result)
}

func (a *App) sunspecRead(args []string) error {
	fs := a.newFlagSet("sunspec read")
	common := commonOptions{}
	addCommonFlags(fs, &common, output.FormatTable, "output format: table, text, json, or csv", true, true)
	modelID := fs.Uint("model", 0, "SunSpec model id")
	pointID := fs.String("point", "", "SunSpec point id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *modelID == 0 || *modelID > 65535 || strings.TrimSpace(*pointID) == "" {
		return fmt.Errorf("%w: --model and --point are required", modbusclient.ErrValidation)
	}
	cfg, format, err := common.loadConfig(fs, output.FormatTable)
	if err != nil {
		return err
	}
	if err := validateSnapshotFormat(format); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout)
	defer cancel()
	client, err := a.openClient(ctx, cfg, common)
	if err != nil {
		return err
	}
	defer client.Close()
	result, err := sunspec.NewScanner(client).ReadPoint(ctx, uint16(*modelID), *pointID)
	if err != nil {
		return err
	}
	return a.renderSunSpecRead(format, result)
}

func (a *App) openClient(ctx context.Context, cfg config.Config, common commonOptions) (modbusclient.Client, error) {
	if common.verbose {
		fmt.Fprintf(a.err, "verbose: connect transport=%s address=%s unit_id=%d timeout=%s\n", cfg.Connection.Transport, cfg.Connection.Address, cfg.Connection.UnitID, cfg.Connection.Timeout)
	}
	client, err := a.factory.New(cfg, modbusclient.Options{Verbose: common.verbose, Debug: common.debug})
	if err != nil {
		return nil, err
	}
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

func printReadHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage of read:
  modbus-cli read coils [flags]
  modbus-cli read discrete-inputs [flags]
  modbus-cli read holding-registers [flags]
  modbus-cli read input-registers [flags]

Examples:
  modbus-cli read coils --address 0 --quantity 8
  modbus-cli read holding-registers --address 0 --quantity 2 --type float32`)
}

func printWriteHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage of write:
  modbus-cli write coil [flags]
  modbus-cli write coils [flags]
  modbus-cli write register [flags]
  modbus-cli write registers [flags]

Writes are dry-run by default. Add --yes to send.

Examples:
  modbus-cli write coil --address 0 --value on
  modbus-cli write register --address 10 --type uint16 --value 42 --yes
  modbus-cli write registers --address 20 --type float32 --value 12.5 --yes`)
}

func printWatchHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage of watch:
  modbus-cli watch coils [flags]
  modbus-cli watch discrete-inputs [flags]
  modbus-cli watch holding-registers [flags]
  modbus-cli watch input-registers [flags]

Examples:
  modbus-cli watch coils --address 0 --quantity 8 --interval 1s
  modbus-cli watch holding-registers --address 0 --quantity 2 --type float32 --format jsonl`)
}

func printPointHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage of named point commands:
  modbus-cli points [flags]
  modbus-cli read-point <name> [flags]
  modbus-cli write-point <name> --value <value> [flags]
  modbus-cli watch-point <name> [flags]

Examples:
  modbus-cli points
  modbus-cli read-point active_power
  modbus-cli write-point breaker_closed --value on --yes
  modbus-cli watch-point active_power --interval 1s --format jsonl`)
}

func printSunSpecHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage of sunspec:
  modbus-cli sunspec scan [flags]
  modbus-cli sunspec models [flags]
  modbus-cli sunspec read --model <id> --point <id> [flags]

Examples:
  modbus-cli sunspec scan
  modbus-cli sunspec models --format json
  modbus-cli sunspec read --model 1 --point Mn`)
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func anyValues(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func identifyObjectName(id byte) string {
	switch id {
	case 0:
		return "vendor_name"
	case 1:
		return "product_code"
	case 2:
		return "major_minor_revision"
	case 3:
		return "vendor_url"
	case 4:
		return "product_name"
	case 5:
		return "model_name"
	case 6:
		return "user_application_name"
	default:
		return "object_" + strconv.Itoa(int(id))
	}
}
