// Package sunspec provides SunSpec-specific discovery over the CLI's
// context-aware Modbus wrapper.
package sunspec

import (
	"context"
	"errors"
	"fmt"

	gosunspec "github.com/andig/gosunspec"
	"github.com/andig/gosunspec/layout"
	_ "github.com/andig/gosunspec/models"
	"github.com/andig/gosunspec/spi"

	"github.com/DishanRajapaksha/modbus-cli/internal/modbusclient"
)

type Scanner struct {
	client modbusclient.Client
}

func NewScanner(client modbusclient.Client) *Scanner {
	return &Scanner{client: client}
}

type ScanResult struct {
	Devices []Device `json:"devices"`
}

type Device struct {
	Index  int     `json:"index"`
	Models []Model `json:"models"`
}

type Model struct {
	ID     uint16  `json:"id"`
	Blocks []Block `json:"blocks"`
}

type Block struct {
	Index   int     `json:"index"`
	Address uint16  `json:"address"`
	Length  uint16  `json:"length"`
	Points  []Point `json:"points,omitempty"`
}

type Point struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Address uint16 `json:"address"`
	Length  uint16 `json:"length"`
}

type ReadResult struct {
	Model uint16 `json:"model"`
	Point string `json:"point"`
	Value any    `json:"value"`
	Type  string `json:"type"`
}

func (s *Scanner) Scan(ctx context.Context) (ScanResult, error) {
	array, err := (&layout.SunSpecLayout{}).Open(newDriver(ctx, s.client))
	if err != nil {
		return ScanResult{}, err
	}
	return summarize(array), nil
}

func (s *Scanner) ReadPoint(ctx context.Context, modelID uint16, pointID string) (ReadResult, error) {
	array, err := (&layout.SunSpecLayout{}).Open(newDriver(ctx, s.client))
	if err != nil {
		return ReadResult{}, err
	}
	model, err := findModel(array, gosunspec.ModelId(modelID))
	if err != nil {
		return ReadResult{}, err
	}
	var found gosunspec.Point
	var readErr error
	model.Do(func(block gosunspec.Block) {
		if found != nil || readErr != nil {
			return
		}
		point, err := block.Point(pointID)
		if err != nil {
			return
		}
		if err := block.Read(pointID); err != nil {
			readErr = err
			return
		}
		found = point
	})
	if readErr != nil {
		return ReadResult{}, readErr
	}
	if found == nil {
		return ReadResult{}, fmt.Errorf("%w: point %q", gosunspec.ErrNoSuchPoint, pointID)
	}
	return ReadResult{Model: modelID, Point: pointID, Type: found.Type(), Value: safePointValue(found)}, nil
}

func summarize(array gosunspec.Array) ScanResult {
	result := ScanResult{}
	deviceIndex := 0
	array.Do(func(device gosunspec.Device) {
		outDevice := Device{Index: deviceIndex}
		deviceIndex++
		device.Do(func(model gosunspec.Model) {
			outModel := Model{ID: uint16(model.Id())}
			blockIndex := 0
			model.Do(func(block gosunspec.Block) {
				blockSPI, ok := block.(spi.BlockSPI)
				if !ok {
					return
				}
				outBlock := Block{Index: blockIndex, Address: anchorAddress(blockSPI.Anchor()), Length: blockSPI.Length()}
				blockIndex++
				block.Do(func(point gosunspec.Point) {
					pointSPI, ok := point.(spi.PointSPI)
					if !ok {
						return
					}
					outBlock.Points = append(outBlock.Points, Point{
						ID:      point.Id(),
						Type:    point.Type(),
						Address: outBlock.Address + pointSPI.Offset(),
						Length:  pointSPI.Length(),
					})
				})
				outModel.Blocks = append(outModel.Blocks, outBlock)
			})
			outDevice.Models = append(outDevice.Models, outModel)
		})
		result.Devices = append(result.Devices, outDevice)
	})
	return result
}

func findModel(array gosunspec.Array, id gosunspec.ModelId) (gosunspec.Model, error) {
	var found gosunspec.Model
	array.Do(func(device gosunspec.Device) {
		if found != nil {
			return
		}
		models := device.Collect(gosunspec.SameModelId(id))
		if len(models) > 0 {
			found = models[0]
		}
	})
	if found == nil {
		return nil, gosunspec.ErrNoSuchModel
	}
	return found, nil
}

func anchorAddress(anchor spi.Anchor) uint16 {
	if value, ok := anchor.(uint16); ok {
		return value
	}
	return 0
}

func safePointValue(point gosunspec.Point) (value any) {
	defer func() {
		if recover() != nil {
			value = nil
		}
	}()
	if point.Error() != nil {
		return nil
	}
	return point.Value()
}

type driver struct {
	ctx    context.Context
	client modbusclient.Client
}

func newDriver(ctx context.Context, client modbusclient.Client) *driver {
	return &driver{ctx: ctx, client: client}
}

func (d *driver) BaseOffsets() []uint16 {
	return []uint16{40000, 50000, 0}
}

func (d *driver) ReadWords(address uint16, length uint16) ([]byte, error) {
	return d.client.ReadHoldingRegisters(d.ctx, address, length)
}

func (d *driver) Read(block spi.BlockSPI, pointIDs ...string) error {
	points, err := block.Plan(pointIDs...)
	if err != nil {
		return err
	}
	for _, point := range points {
		address, err := pointAddress(block, point)
		if err != nil {
			point.SetError(err)
			return err
		}
		data, err := d.ReadWords(address, point.Length())
		if err != nil {
			point.SetError(err)
			return err
		}
		if err := point.Unmarshal(data); err != nil {
			point.SetError(err)
			return err
		}
	}
	return nil
}

func (d *driver) Write(spi.BlockSPI, ...string) error {
	return errors.New("sunspec writes are not implemented")
}

func pointAddress(block spi.BlockSPI, point spi.PointSPI) (uint16, error) {
	base, ok := block.Anchor().(uint16)
	if !ok {
		return 0, errors.New("sunspec block has no uint16 address anchor")
	}
	return base + point.Offset(), nil
}
