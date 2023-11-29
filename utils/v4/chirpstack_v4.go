//go:build chirpstack4
// +build chirpstack4

package v4

import (
	"context"
	"fmt"

	"github.com/chirpstack/chirpstack/api/go/v4/api"
	csCommon "github.com/chirpstack/chirpstack/api/go/v4/common"
	"github.com/edgexfoundry/device-lora-go/config"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	Limit uint32 = 100
)

func Init(conn *grpc.ClientConn, config config.ChirpStackConfig) (tenantId, appId string, err error) {
	// 登录chirpstack
	ctx := Login(conn, config.Username, config.Password)
	// 获取tentant
	tenantId, err = initTenant(conn, ctx)
	// 获取application
	appId, err = initApplication(conn, ctx, tenantId)
	return
}

func Login(conn *grpc.ClientConn, username, password string) (ctx context.Context) {
	client := api.NewInternalServiceClient(conn)
	if resp, err := client.Login(context.Background(), &api.LoginRequest{
		Email:    username,
		Password: password,
	}); err == nil {
		fmt.Println("jwt", resp.Jwt)
		ctx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+resp.Jwt))
	}
	return
}

func initTenant(conn *grpc.ClientConn, ctx context.Context) (id string, err error) {
	client := api.NewTenantServiceClient(conn)
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

func initApplication(conn *grpc.ClientConn, ctx context.Context, tenantId string) (id string, err error) {
	client := api.NewApplicationServiceClient(conn)

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

func CreateProfile(conn *grpc.ClientConn, ctx context.Context, tenantId string, name string, codec string) (id string, err error) {
	client := api.NewDeviceProfileServiceClient(conn)
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

func DeleteProfile(conn *grpc.ClientConn, ctx context.Context, tenantId string, name string) (err error) {
	client := api.NewDeviceProfileServiceClient(conn)
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

func CreateGateway(conn *grpc.ClientConn, ctx context.Context, gateWayId string, name string, tenantId string) (err error) {
	client := api.NewGatewayServiceClient(conn)

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
			TenantId:      tenantId,
			StatsInterval: 3000,
		},
	}); err != nil {
		fmt.Println("gateway create fail", err)
	} else {
		fmt.Println("gateway create success")
	}
	return
}

func UpdateGateway(conn *grpc.ClientConn, ctx context.Context, gateWayId string, name string) (err error) {
	client := api.NewGatewayServiceClient(conn)

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

func DeleteGateway(conn *grpc.ClientConn, ctx context.Context, gateWayId string) (err error) {
	client := api.NewGatewayServiceClient(conn)
	if _, err = client.Delete(ctx, &api.DeleteGatewayRequest{
		GatewayId: gateWayId,
	}); err == nil {
		fmt.Println("gateway delete success")
	} else {
		fmt.Println("gateway delete fail", err)
	}
	return
}

func CreateDevice(conn *grpc.ClientConn, ctx context.Context, DevEUI string, name string, deviceProfileId string, applicationId string) (err error) {
	client := api.NewDeviceServiceClient(conn)
	if _, err = client.Create(ctx, &api.CreateDeviceRequest{
		Device: &api.Device{
			DevEui:          DevEUI,
			Name:            name,
			ApplicationId:   applicationId,
			DeviceProfileId: deviceProfileId,
			SkipFcntCheck:   true,
		},
	}); err == nil {
		fmt.Println("dev create success")

	} else {
		fmt.Println("dev create fail", err)
	}

	return
}

func ActivateDevice(conn *grpc.ClientConn, ctx context.Context, DevEUI string, key string) (err error) {
	client := api.NewDeviceServiceClient(conn)
	var resp *api.GetRandomDevAddrResponse
	if resp, err = client.GetRandomDevAddr(ctx, &api.GetRandomDevAddrRequest{
		DevEui: DevEUI,
	}); err != nil {
		fmt.Println("dev get addr fail", err)
	} else {
		fmt.Println("dev get addr", resp.DevAddr)
		if _, err = client.Activate(ctx, &api.ActivateDeviceRequest{
			DeviceActivation: &api.DeviceActivation{
				DevEui:      DevEUI,
				DevAddr:     resp.DevAddr,
				AppSKey:     key,
				NwkSEncKey:  key,
				SNwkSIntKey: key,
				FNwkSIntKey: key,
			},
		}); err != nil {
			fmt.Println("dev Activate fail", err)
		} else {
			fmt.Println("dev activate success")
		}
	}
	return
}

func UpdateDevice(conn *grpc.ClientConn, ctx context.Context, DevEUI string, name string) (err error) {
	client := api.NewDeviceServiceClient(conn)

	var resp *api.GetDeviceResponse
	if resp, err = client.Get(ctx, &api.GetDeviceRequest{
		DevEui: DevEUI,
	}); err == nil && resp.Device != nil {
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

func DeleteDevice(conn *grpc.ClientConn, ctx context.Context, deviceName string, DevEUI string) (err error) {
	client := api.NewDeviceServiceClient(conn)
	if _, err = client.Delete(ctx, &api.DeleteDeviceRequest{
		DevEui: DevEUI,
	}); err == nil {
		fmt.Println("dev delete success")
	} else {
		fmt.Println("dev delete fail", err)
	}
	return
}
