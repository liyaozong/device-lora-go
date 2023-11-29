//go:build chirpstack4
// +build chirpstack4

package driver

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/chirpstack/chirpstack/api/go/v4/api"
	"github.com/edgexfoundry/device-lora-go/config"
	sdkModel "github.com/edgexfoundry/device-sdk-go/v3/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/models"
)

type Listener struct {
	driver     *LoraDriver
	config     config.ChirpStackConfig
	DeviceName string
	Stop       bool
	stream     api.InternalService_StreamDeviceEventsClient
}

func (e *Listener) Listening(chirp *ChirpStack, ctx context.Context, DevEUI string) (err error) {
	var device models.Device
	if device, err = e.driver.sdk.GetDeviceByName(e.DeviceName); err != nil {
		return err
	}

	var ok bool
	var deviceResource models.DeviceResource
	var profile models.DeviceProfile
	if profile, err = e.driver.sdk.GetProfileByName(device.ProfileName); err == nil {
		// lorawan返回的是json对象数据，
		if len(profile.DeviceResources) == 1 {
			sourceName := profile.DeviceResources[0].Name
			optional := profile.DeviceResources[0].Properties.Optional
			if _, ok = optional[CODEC]; ok {
				deviceResource, ok = e.driver.sdk.DeviceResource(e.DeviceName, sourceName)
			}
		}
	}

	if !ok {
		return errors.New("device resource not found")
	}

	client := api.NewInternalServiceClient(chirp.conn)
	if e.stream, err = client.StreamDeviceEvents(ctx, &api.StreamDeviceEventsRequest{
		DevEui: DevEUI,
	}); err == nil {
		for {
			resp, err := e.stream.Recv()
			if err == io.EOF {
				break
			}

			if e.Stop {
				break
			}

			if err != nil {
				fmt.Printf("[listener] recv a err %v", err)
				break
			}

			var commandValues []*sdkModel.CommandValue
			commandValue, err := e.driver.NewResult(deviceResource, resp.Body)
			if err != nil {
				fmt.Printf("[listener] Incoming data ignored: %v", err)
				continue
			}
			commandValues = append(commandValues, commandValue)

			asyncValues := &sdkModel.AsyncValues{
				DeviceName:    e.DeviceName,
				SourceName:    deviceResource.Name,
				CommandValues: commandValues,
			}

			fmt.Printf("[listener] Incoming reading received: device=%v msg=%v", e.DeviceName, resp.Body)

			e.driver.AsyncCh <- asyncValues
		}
	}
	return
}

func (e *Listener) Cancel() {
	e.Stop = true
	e.stream.Context().Done()
}