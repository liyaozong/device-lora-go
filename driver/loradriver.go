//

// Copyright (c) 2023 Starblaze Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
//
// CONTRIBUTORS              COMPANY
//===============================================================
// 1. Yaozong.li             Starblaze
//

package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/edgexfoundry/device-lora-go/config"
	"github.com/edgexfoundry/device-sdk-go/v3/pkg/interfaces"
	sdkModels "github.com/edgexfoundry/device-sdk-go/v3/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/models"
)

type LoraDriver struct {
	sdk       interfaces.DeviceServiceSDK
	logger    logger.LoggingClient
	AsyncCh   chan<- *sdkModels.AsyncValues
	chirp     ChirpStack
	listeners map[string]Listener
}

func (driver *LoraDriver) Initialize(sdk interfaces.DeviceServiceSDK) (err error) {
	driver.sdk = sdk
	driver.logger = sdk.LoggingClient()
	driver.AsyncCh = sdk.AsyncValuesChannel()
	driver.listeners = make(map[string]Listener)

	serviceConfig := &config.ServiceConfig{}

	if err = sdk.LoadCustomConfig(serviceConfig, "ChirpStack"); err != nil {
		return fmt.Errorf("unable to load 'ChirpStack' custom configuration: %s", err.Error())
	}

	driver.logger.Infof("Custom config is: %v", serviceConfig.ChirpStack)

	if err = serviceConfig.ChirpStack.Validate(); err != nil {
		return fmt.Errorf("'ChirpStack' custom configuration validation failed: %s", err.Error())
	}

	driver.chirp = ChirpStack{
		config: serviceConfig.ChirpStack,
	}

	err = driver.chirp.Init()
	return err
}

func (driver *LoraDriver) Start() (err error) {
	devices := driver.sdk.Devices()
	// 登录chirpstack
	var ctx context.Context
	if ctx, err = driver.chirp.Login(); err != nil {
		return
	}

	for _, device := range devices {
		if protocolParams, err := getDeviceParameters(device.Protocols); err == nil && !protocolParams.Gateway {
			//监听设备
			listener := Listener{
				driver:     driver,
				DeviceName: device.Name,
				config:     driver.chirp.config,
				Stop:       false,
			}
			driver.listeners[protocolParams.EUI] = listener
			go listener.Listening(&driver.chirp, ctx, protocolParams.EUI)
		}
	}
	handler := NewLoraHandler(driver.sdk)
	return handler.Start()
}

func (driver *LoraDriver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModels.CommandRequest) (responses []*sdkModels.CommandValue, err error) {
	driver.logger.Info("Lora not support HandleReadCommands function")

	return nil, fmt.Errorf("Lora not support HandleReadCommands function")
}

func (driver *LoraDriver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModels.CommandRequest, params []*sdkModels.CommandValue) (err error) {
	var protocolParams LoraProtocolParams
	if protocolParams, err = getDeviceParameters(protocols); err != nil {
		return fmt.Errorf("Device parameters missing :%s \n", err.Error())
	}

	for i, req := range reqs {
		// First get device resource instance, needed during validation of the
		// data received in the write command request
		// RunningService returns the Service instance which is running
		// service.DeviceResource retrieves the specific DeviceResource instance
		// from cache according to the Device name and Device Resource name
		deviceResource, ok := driver.sdk.DeviceResource(deviceName, req.DeviceResourceName)
		if !ok {
			return fmt.Errorf("Incoming Writing ignored. Resource '%s' not found", req.DeviceResourceName)
		}

		// Its time to form payload to be sent to end device.
		// For this fisrt get the data received in the write command request
		// This data is validated against the expected value type of device resource
		// With the data and uri create new http PUT request
		// And, set the content type header for the PUT request
		reading := params[i].Value
		valueType := deviceResource.Properties.ValueType
		switch valueType {
		case common.ValueTypeObject:
			buf, _ := json.Marshal(reading)
			if !json.Valid([]byte(buf)) {
				return fmt.Errorf("PUT request data is invalid JSON string")
			}
		case common.ValueTypeBool, common.ValueTypeString, common.ValueTypeUint8,
			common.ValueTypeUint16, common.ValueTypeUint32, common.ValueTypeUint64,
			common.ValueTypeInt8, common.ValueTypeInt16, common.ValueTypeInt32,
			common.ValueTypeInt64, common.ValueTypeFloat32, common.ValueTypeFloat64:
			// All other types
			contentType := common.ContentTypeText
			_, err = validateCommandValue(deviceResource, reading, deviceResource.Properties.ValueType, contentType)
			if err != nil {
				// handle error
				return fmt.Errorf("PUT request data is not valid")
			}

		default:
			return fmt.Errorf("Unsupported value type: %v", valueType)
		}

		driver.logger.Debugf("Send command to %s", protocolParams.EUI)
	}

	return nil
}

func getDeviceParameters(protocols map[string]models.ProtocolProperties) (LoraProtocolParams, error) {
	var restDeviceProtocolParams LoraProtocolParams
	protocolParams, paramsExists := protocols[LoraProtocol]
	if !paramsExists {
		return restDeviceProtocolParams, errors.New("No End device parameters defined in the protocol list")
	}

	// Get end device EUI
	if host, ok := protocolParams[LoraEUI]; ok {
		if restDeviceProtocolParams.EUI, ok = host.(string); !ok {
			return restDeviceProtocolParams, errors.New("EUI is not string type")
		}
	} else {
		return restDeviceProtocolParams, errors.New("EUI not found")
	}

	// Get end device LoraGateway
	if gateway, ok := protocolParams[LoraGateway]; ok {
		if restDeviceProtocolParams.Gateway, ok = gateway.(bool); !ok {
			return restDeviceProtocolParams, errors.New("LoraGateway is not string type")
		}
	} else {
		return restDeviceProtocolParams, errors.New("LoraGateway not found")
	}

	return restDeviceProtocolParams, nil
}

func (driver *LoraDriver) Stop(force bool) error {
	driver.logger.Debugf("RestDriver.Stop called: force=%v", force)
	return nil
}

func (driver *LoraDriver) AddDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) (err error) {
	driver.logger.Info("AddDevice %s", deviceName)
	var protocolParams LoraProtocolParams
	if protocolParams, err = getDeviceParameters(protocols); err != nil {
		return fmt.Errorf("Device parameters missing :%s \n", err.Error())
	}

	var device models.Device
	if device, err = driver.sdk.GetDeviceByName(deviceName); err != nil {
		return
	}

	var profile models.DeviceProfile
	if profile, err = driver.sdk.GetProfileByName(device.ProfileName); err != nil {
		return
	}

	err = driver.AddLoraDevice(&driver.chirp, device, profile, protocolParams)

	return
}

func (driver *LoraDriver) UpdateDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) (err error) {
	driver.logger.Info("UpdateDevice %s", deviceName)
	var protocolParams LoraProtocolParams
	if protocolParams, err = getDeviceParameters(protocols); err != nil {
		return fmt.Errorf("Device parameters missing :%s \n", err.Error())
	}

	var device models.Device
	if device, err = driver.sdk.GetDeviceByName(deviceName); err != nil {
		return err
	}

	err = driver.UpdateLoraDevice(&driver.chirp, device, protocolParams)

	return
}

func (driver *LoraDriver) RemoveDevice(deviceName string, protocols map[string]models.ProtocolProperties) (err error) {
	driver.logger.Info("RemoveDevice %s", deviceName)
	var protocolParams LoraProtocolParams
	if protocolParams, err = getDeviceParameters(protocols); err != nil {
		return fmt.Errorf("Device parameters missing :%s \n", err.Error())
	}

	err = driver.RemoveLoraDevice(&driver.chirp, deviceName, protocolParams)

	return
}

func (driver *LoraDriver) Discover() error {
	return fmt.Errorf("driver's Discover function isn't implemented")
}

func (driver *LoraDriver) ValidateDevice(device models.Device) error {
	if _, ok := device.Protocols[LoraProtocol]; ok {
		_, err := getDeviceParameters(device.Protocols)
		if err != nil {
			return fmt.Errorf("invalid protocol properties, %v", err)
		}
	}
	return nil
}

func (driver *LoraDriver) NewResult(resource models.DeviceResource, reading interface{}) (*sdkModels.CommandValue, error) {
	var err error
	var result = &sdkModels.CommandValue{}

	valueType := resource.Properties.ValueType

	var val interface{}
	switch valueType {
	case common.ValueTypeObject:
		val = reading
	default:
		return nil, fmt.Errorf("return result fail, none supported value type: %v", valueType)
	}

	if result, err = sdkModels.NewCommandValue(resource.Name, valueType, val); err != nil {
		return nil, err
	}
	result.Origin = time.Now().UnixNano()

	return result, nil
}
