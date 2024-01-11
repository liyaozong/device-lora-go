//go:build chirpstack3
// +build chirpstack3

package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/brocaar/chirpstack-api/go/v3/as/external/api"
	"github.com/edgexfoundry/device-lora-go/config"
	sdkModels "github.com/edgexfoundry/device-sdk-go/v3/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/models"
)

type PayloadJson struct {
	ApplicationId string    `json:"applicationID"`
	DevEUI        string    `json:"devEUI"`
	ObjectJSON    string    `json:"objectJSON"`
	PublishedAt   time.Time `json:"publishedAt"`
}

type Listener struct {
	driver     *LoraDriver
	config     config.ChirpStackConfig
	DeviceName string
	Stop       bool
	stream     api.DeviceService_StreamEventLogsClient
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

	client := api.NewDeviceServiceClient(chirp.conn)
	if e.stream, err = client.StreamEventLogs(ctx, &api.StreamDeviceEventLogsRequest{
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

			// 没有收到有用数据，跳过执行
			if resp == nil || resp.Type != "up" {
				continue
			}

			var payloadJson PayloadJson
			err = json.Unmarshal([]byte(resp.PayloadJson), &payloadJson)
			if err != nil {
				continue
			}

			var commandValues []*sdkModels.CommandValue
			commandValue, err := e.driver.NewResult(deviceResource, payloadJson.ObjectJSON)
			if err != nil {
				continue
			}
			commandValues = append(commandValues, commandValue)

			asyncValues := &sdkModels.AsyncValues{
				DeviceName:    e.DeviceName,
				SourceName:    deviceResource.Name,
				CommandValues: commandValues,
			}

			fmt.Printf("[listener] Incoming reading received: device=%v msg=%v", e.DeviceName, resp.PayloadJson)

			e.driver.AsyncCh <- asyncValues
		}
	}
	return
}

func (e *Listener) Cancel() {
	e.Stop = true
	if e.stream != nil {
		e.stream.Context().Done()
	}
}
