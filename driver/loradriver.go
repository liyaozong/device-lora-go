//
// Copyright (c) 2019 Intel Corporation
// Copyright (c) 2023 IOTech Ltd
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
// 1. Sathya Durai           HCL Technologies
// 2. Sudhamani Bijivemula   HCL Technologies
// 3. Vijay Annamalaisamy    HCL Technologies
//

package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chirpstack/chirpstack/api/go/v4/api"
	csCommon "github.com/chirpstack/chirpstack/api/go/v4/common"
	"github.com/edgexfoundry/device-sdk-go/v3/pkg/interfaces"
	sdkModel "github.com/edgexfoundry/device-sdk-go/v3/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/models"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	Limit    uint32 = 100
	Key      string = "bc67cd6eb45a08d975050b1887b93c23"
	Host     string = "172.16.64.157:8082"
	Admin    string = "admin"
	Password string = "admin"
)

type LoraDriver struct {
	sdk           interfaces.DeviceServiceSDK
	logger        logger.LoggingClient
	AsyncCh       chan<- *sdkModel.AsyncValues
	conn          *grpc.ClientConn
	tenantId      string
	applicationId string
	// profileId     string //不使用全局profile，为每种设备创建profile
}

func (driver *LoraDriver) Initialize(sdk interfaces.DeviceServiceSDK) (err error) {
	driver.logger = sdk.LoggingClient()
	driver.AsyncCh = sdk.AsyncValuesChannel()
	driver.sdk = sdk
	driver.conn, err = grpc.Dial(Host, grpc.WithInsecure())

	// 登录chirpstack
	ctx := driver.login()
	// 获取tentant
	driver.tenantId, err = driver.initTenant(ctx)
	// 获取application
	driver.applicationId, err = driver.initApplication(ctx, driver.tenantId)
	// 获取device profile
	// driver.profileId, err = driver.createProfile(ctx, driver.tenantId, "Starblaze device profile", "default", "")

	return err
}

func (driver *LoraDriver) Start() error {
	devices := driver.sdk.Devices()
	for _, device := range devices {
		if !strings.Contains(device.ProfileName, LoraGateway) {
			if protocolParams, err := getDeviceParameters(device.Protocols); err == nil {
				// 登录chirpstack
				ctx := driver.login()
				//监听设备，上报数据
				driver.recvDeviceStream(ctx, protocolParams.EUI, device.Name, "json")
			}
		}
	}
	handler := NewLoraHandler(driver.sdk)
	return handler.Start()
}

func (driver *LoraDriver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModel.CommandRequest) (responses []*sdkModel.CommandValue, err error) {
	driver.logger.Info("Lora not support HandleReadCommands function")

	return nil, fmt.Errorf("Lora not support HandleReadCommands function")
}

func (driver *LoraDriver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModel.CommandRequest, params []*sdkModel.CommandValue) (err error) {
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

	// Get end device codec
	if codec, ok := protocolParams[LoraCodec]; ok {
		if restDeviceProtocolParams.Codec, ok = codec.(string); !ok {
			return restDeviceProtocolParams, errors.New("Codec is not string type")
		}
	} else {
		return restDeviceProtocolParams, errors.New("Codec not found")
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
		return err
	}

	// 登录chirpstack
	ctx := driver.login()
	if strings.Contains(device.ProfileName, LoraGateway) {
		// 创建网关
		err = driver.createGateway(ctx, protocolParams.EUI, deviceName)
	} else {
		// 创建设备
		err = driver.createDevice(ctx, protocolParams.EUI, deviceName, device.ProfileName, protocolParams.Codec)
	}

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

	// 登录chirpstack
	ctx := driver.login()
	if strings.Contains(device.ProfileName, LoraGateway) {
		// 更新网关
		err = driver.updateGateway(ctx, protocolParams.EUI, deviceName)
	} else {
		// 更新设备
		err = driver.updateDevice(ctx, protocolParams.EUI, deviceName)
	}

	return
}

func (driver *LoraDriver) RemoveDevice(deviceName string, protocols map[string]models.ProtocolProperties) (err error) {
	driver.logger.Info("RemoveDevice %s", deviceName)
	var protocolParams LoraProtocolParams
	if protocolParams, err = getDeviceParameters(protocols); err != nil {
		return fmt.Errorf("Device parameters missing :%s \n", err.Error())
	}

	// var device models.Device
	// if device, err = driver.sdk.GetDeviceByName(deviceName); err != nil {
	// 	return err
	// }

	// 登录chirpstack
	ctx := driver.login()
	// if strings.Contains(device.ProfileName, LoraGateway) {
	// 删除网关
	err = driver.deleteGateway(ctx, protocolParams.EUI)
	// } else {
	// 删除设备
	err = driver.deleteDevice(ctx, deviceName, protocolParams.EUI)
	// }

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

type APIToken string

func (a APIToken) GetRequestMetadata(ctx context.Context, url ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", a),
	}, nil
}

func (a APIToken) RequireTransportSecurity() bool {
	return false
}

func (driver *LoraDriver) login() (ctx context.Context) {
	client := api.NewInternalServiceClient(driver.conn)
	if resp, err := client.Login(context.Background(), &api.LoginRequest{
		Email:    Admin,
		Password: Password,
	}); err == nil {
		fmt.Println("jwt", resp.Jwt)
		ctx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+resp.Jwt))
	}
	return
}

func (driver *LoraDriver) initTenant(ctx context.Context) (id string, err error) {
	client := api.NewTenantServiceClient(driver.conn)
	var resp *api.ListTenantsResponse
	if resp, err = client.List(ctx, &api.ListTenantsRequest{
		Limit: Limit,
	}); err != nil {
		fmt.Println("tenants", err)
		return "", err
	} else {
		fmt.Println("tenants", resp.Result)
		if len(resp.Result) > 0 {
			return resp.Result[0].Id, nil
		} else {
			var resp *api.CreateTenantResponse
			if resp, err = client.Create(ctx, &api.CreateTenantRequest{
				Tenant: &api.Tenant{
					Id:   uuid.NewV4().String(),
					Name: "Starblaze",
				},
			}); err == nil {
				return resp.Id, nil
			}
		}
	}
	return "", err
}

func (driver *LoraDriver) initApplication(ctx context.Context, tenantId string) (id string, err error) {
	client := api.NewApplicationServiceClient(driver.conn)

	var resp *api.ListApplicationsResponse
	if resp, err = client.List(ctx, &api.ListApplicationsRequest{
		Limit:    Limit,
		TenantId: tenantId,
	}); err != nil {
		fmt.Println("apps", err)
		return "", err
	} else {
		fmt.Println("apps", resp.Result)
		if len(resp.Result) > 0 {
			return resp.Result[0].Id, nil
		} else {
			var resp *api.CreateApplicationResponse
			if resp, err = client.Create(ctx, &api.CreateApplicationRequest{
				Application: &api.Application{
					Id:       uuid.NewV4().String(),
					Name:     "Starblaze App",
					TenantId: tenantId,
				},
			}); err == nil {
				return resp.Id, nil
			}
		}
	}
	return "", err
}

func (driver *LoraDriver) createProfile(ctx context.Context, tenantId string, name string, codec string) (id string, err error) {
	client := api.NewDeviceProfileServiceClient(driver.conn)
	var resp *api.ListDeviceProfilesResponse
	if resp, err = client.List(ctx, &api.ListDeviceProfilesRequest{
		Limit:    Limit,
		TenantId: tenantId,
		Search:   name,
	}); err == nil && resp.Result != nil {
		fmt.Println("profiles", resp.Result)
		if len(resp.Result) > 0 && resp.Result[0] != nil {
			//返回已存在的profile
			return resp.Result[0].Id, nil
		}
	}

	//创建profile
	var resp1 *api.CreateDeviceProfileResponse
	if resp1, err = client.Create(ctx, &api.CreateDeviceProfileRequest{
		DeviceProfile: &api.DeviceProfile{
			Id:                  uuid.NewV4().String(),
			TenantId:            tenantId,
			Name:                name,
			Region:              csCommon.Region_CN470,
			MacVersion:          csCommon.MacVersion_LORAWAN_1_0_2,
			RegParamsRevision:   csCommon.RegParamsRevision_A,
			AdrAlgorithmId:      "default", // options: default, lr_fhss, lora_lr_fhss
			UplinkInterval:      600,
			PayloadCodecScript:  codec,
			PayloadCodecRuntime: api.CodecRuntime_JS,
		},
	}); err == nil {
		return resp1.Id, nil
	}

	return "", err
}

func (driver *LoraDriver) deleteProfile(ctx context.Context, tenantId string, name string) (err error) {
	client := api.NewDeviceProfileServiceClient(driver.conn)
	var resp *api.ListDeviceProfilesResponse
	if resp, err = client.List(ctx, &api.ListDeviceProfilesRequest{
		Limit:    Limit,
		TenantId: tenantId,
		Search:   name,
	}); err == nil && resp.Result != nil {
		for _, profile := range resp.Result {
			if _, err = client.Delete(ctx, &api.DeleteDeviceProfileRequest{
				Id: profile.Id,
			}); err == nil {
				fmt.Println("profile delete success")
			}
		}
	}
	return
}

func (driver *LoraDriver) createGateway(ctx context.Context, gateWayId string, name string) (err error) {
	client := api.NewGatewayServiceClient(driver.conn)

	var resp *api.GetGatewayResponse
	if resp, err = client.Get(ctx, &api.GetGatewayRequest{
		GatewayId: gateWayId,
	}); err == nil && resp.Gateway != nil {
		fmt.Println("gateway is exist")
		return
	}

	if _, err = client.Create(ctx, &api.CreateGatewayRequest{
		Gateway: &api.Gateway{
			GatewayId:     gateWayId,
			Name:          name,
			TenantId:      driver.tenantId,
			StatsInterval: 3000,
		},
	}); err != nil {
		fmt.Println("gateway create fail", err)
	} else {
		fmt.Println("gateway create success")
	}
	return
}

func (driver *LoraDriver) updateGateway(ctx context.Context, gateWayId string, name string) (err error) {
	client := api.NewGatewayServiceClient(driver.conn)

	var resp *api.GetGatewayResponse
	if resp, err = client.Get(ctx, &api.GetGatewayRequest{
		GatewayId: gateWayId,
	}); err == nil && resp.Gateway != nil {
		if _, err := client.Update(ctx, &api.UpdateGatewayRequest{
			Gateway: &api.Gateway{
				GatewayId:     gateWayId,
				Name:          name,
				TenantId:      resp.Gateway.TenantId,
				StatsInterval: 3000,
			},
		}); err != nil {
			fmt.Println("gateway update fail", err)
		} else {
			fmt.Println("gateway update success")
		}
	} else {
		fmt.Println("dev isn't exist")
	}

	return
}

func (driver *LoraDriver) deleteGateway(ctx context.Context, gateWayId string) (err error) {
	client := api.NewGatewayServiceClient(driver.conn)
	if _, err = client.Delete(ctx, &api.DeleteGatewayRequest{
		GatewayId: gateWayId,
	}); err == nil {
		fmt.Println("gateway delete success")
	} else {
		fmt.Println("gateway delete fail", err)
	}
	return
}

func (driver *LoraDriver) createDevice(ctx context.Context, DevEUI string, name string, profileName string, codec string) (err error) {
	var deviceProfileId string
	// 以设备名称为参数创建设备profile，
	if deviceProfileId, err = driver.createProfile(ctx, driver.tenantId, name, codec); err == nil {
		client := api.NewDeviceServiceClient(driver.conn)
		if _, err = client.Create(ctx, &api.CreateDeviceRequest{
			Device: &api.Device{
				DevEui:          DevEUI,
				Name:            name,
				ApplicationId:   driver.applicationId,
				DeviceProfileId: deviceProfileId,
				SkipFcntCheck:   true,
			},
		}); err == nil {
			fmt.Println("dev create success")

			//激活设备
			var resp *api.GetRandomDevAddrResponse
			if resp, err = client.GetRandomDevAddr(ctx, &api.GetRandomDevAddrRequest{
				DevEui: DevEUI,
			}); err != nil {
				fmt.Println("dev get addr fail", err)
			} else {
				fmt.Println("dev get addr", resp.DevAddr)
				if _, err := client.Activate(ctx, &api.ActivateDeviceRequest{
					DeviceActivation: &api.DeviceActivation{
						DevEui:      DevEUI,
						DevAddr:     resp.DevAddr,
						AppSKey:     Key,
						NwkSEncKey:  Key,
						SNwkSIntKey: Key,
						FNwkSIntKey: Key,
					},
				}); err != nil {
					fmt.Println("dev Activate fail", err)
				} else {
					fmt.Println("dev activate success")
				}
			}

			//监听设备，上报数据
			driver.recvDeviceStream(ctx, DevEUI, name, "json")
		} else {
			fmt.Println("dev create fail", err)
		}
	} else {
		fmt.Println("dev profile create fail", err)
	}

	return
}

func (driver *LoraDriver) updateDevice(ctx context.Context, DevEUI string, name string) (err error) {
	client := api.NewDeviceServiceClient(driver.conn)

	var resp *api.GetDeviceResponse
	if resp, err = client.Get(ctx, &api.GetDeviceRequest{}); err == nil && resp.Device != nil {
		if _, err := client.Update(ctx, &api.UpdateDeviceRequest{
			Device: &api.Device{
				DevEui:          DevEUI,
				Name:            name,
				ApplicationId:   resp.Device.ApplicationId,
				DeviceProfileId: resp.Device.DeviceProfileId,
				SkipFcntCheck:   true,
			},
		}); err != nil {
			fmt.Println("dev update fail", err)
		} else {
			fmt.Println("dev update success")
		}
	} else {
		fmt.Println("dev isn't exist")
	}
	return
}

func (driver *LoraDriver) deleteDevice(ctx context.Context, deviceName string, DevEUI string) (err error) {
	client := api.NewDeviceServiceClient(driver.conn)
	if _, err = client.Delete(ctx, &api.DeleteDeviceRequest{
		DevEui: DevEUI,
	}); err == nil {
		fmt.Println("dev delete success")

		//删除设备profile
		driver.deleteProfile(ctx, driver.tenantId, deviceName)
	} else {
		fmt.Println("dev delete fail", err)
	}
	return
}

func (driver *LoraDriver) recvDeviceStream(ctx context.Context, DevEUI string, deviceName string, sourceName string) {
	client := api.NewInternalServiceClient(driver.conn)
	if stream, err := client.StreamDeviceEvents(ctx, &api.StreamDeviceEventsRequest{
		DevEui: DevEUI,
	}); err == nil {
		go func() {
			fmt.Println("start listener device", DevEUI)
			for {
				if resp, err := stream.Recv(); err == nil {
					var commandValues []*sdkModel.CommandValue
					if deviceResource, ok := driver.sdk.DeviceResource(deviceName, sourceName); ok {
						commandValue, err := driver.newResult(deviceResource, resp.Body)
						if err != nil {
							driver.logger.Errorf("[listener] Incoming data ignored: %v", err)
							continue
						}
						commandValues = append(commandValues, commandValue)
					} else {
						driver.logger.Errorf("[listener] device source not found: device=%v source=%v", deviceName, sourceName)
						continue
					}

					asyncValues := &sdkModel.AsyncValues{
						DeviceName:    deviceName,
						SourceName:    sourceName,
						CommandValues: commandValues,
					}

					driver.logger.Debugf("[listener] Incoming reading received: device=%v msg=%v", deviceName, resp.Body)

					driver.AsyncCh <- asyncValues
				} else {
					fmt.Println("dev recv stream fail", err)
				}
			}
		}()
	} else {
		fmt.Println("dev listener fail", err)
	}
}

func (driver *LoraDriver) newResult(resource models.DeviceResource, reading interface{}) (*sdkModel.CommandValue, error) {
	var err error
	var result = &sdkModel.CommandValue{}

	valueType := resource.Properties.ValueType

	var val interface{}
	switch valueType {
	case common.ValueTypeObject:
		val = reading
	default:
		return nil, fmt.Errorf("return result fail, none supported value type: %v", valueType)
	}

	if result, err = sdkModel.NewCommandValue(resource.Name, valueType, val); err != nil {
		return nil, err
	}
	result.Origin = time.Now().UnixNano()

	return result, nil
}
