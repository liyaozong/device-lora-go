//go:build chirpstack3
// +build chirpstack3

package v3

import (
	"context"
	"fmt"

	"github.com/brocaar/chirpstack-api/go/v3/as/external/api"
	"github.com/edgexfoundry/device-lora-go/config"
	"github.com/golang/protobuf/ptypes/duration"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	Limit64 int64 = 100
)

func Init(conn *grpc.ClientConn, config config.ChirpStackConfig) (netId, orgId, appId int64, err error) {
	// 登录chirpstack
	ctx := Login(conn, config.Username, config.Password)
	// 获取netWorkServer
	netId, err = initNetWorkServer(conn, ctx)
	// 获取organization
	orgId, err = initOrganization(conn, ctx)
	// 获取application
	appId, err = initApplication(conn, ctx, orgId)
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

func initOrganization(conn *grpc.ClientConn, ctx context.Context) (organizationId int64, err error) {
	// 返回值为 ApplicationServiceClient
	client := api.NewOrganizationServiceClient(conn)
	req := api.ListOrganizationRequest{
		Limit: Limit64,
	}
	if resp, err := client.List(ctx, &req); err == nil {
		if len(resp.Result) > 0 {
			return resp.Result[0].Id, nil
		} else {
			var resp *api.CreateOrganizationResponse
			if resp, err = client.Create(ctx, &api.CreateOrganizationRequest{
				Organization: &api.Organization{
					Name: "Starblaze",
				},
			}); err == nil {
				return resp.Id, nil
			}
		}
	}
	return -1, nil
}

func initNetWorkServer(conn *grpc.ClientConn, ctx context.Context) (netWorkServerId int64, err error) {
	client := api.NewNetworkServerServiceClient(conn)
	req := api.ListNetworkServerRequest{
		Limit: Limit64,
	}
	if resp, err := client.List(ctx, &req); err == nil {
		if len(resp.Result) > 0 {
			return resp.Result[0].Id, nil
		} else {
			var resp *api.CreateNetworkServerResponse
			if resp, err = client.Create(ctx, &api.CreateNetworkServerRequest{
				NetworkServer: &api.NetworkServer{
					Name: "Starblaze",
				},
			}); err == nil {
				return resp.Id, nil
			}
		}
	}
	return -1, nil
}

func initApplication(conn *grpc.ClientConn, ctx context.Context, organizationId int64) (id int64, err error) {
	client := api.NewApplicationServiceClient(conn)

	var resp *api.ListApplicationResponse
	if resp, err = client.List(ctx, &api.ListApplicationRequest{
		Limit:          Limit64,
		OrganizationId: organizationId,
	}); err != nil {
		fmt.Println("apps", err)
		return -1, err
	} else {
		fmt.Println("apps", resp.Result)
		if len(resp.Result) > 0 {
			return resp.Result[0].Id, nil
		} else {
			var resp *api.CreateApplicationResponse
			if resp, err = client.Create(ctx, &api.CreateApplicationRequest{
				Application: &api.Application{
					Name:           "Starblaze App",
					OrganizationId: organizationId,
				},
			}); err == nil {
				return resp.Id, nil
			}
		}
	}
	return -1, err
}

func CreateProfile(conn *grpc.ClientConn, ctx context.Context, netId int64, orgId int64, appId int64, name string, codec string) (id string, err error) {
	client := api.NewDeviceProfileServiceClient(conn)
	var resp *api.ListDeviceProfileResponse
	if resp, err = client.List(ctx, &api.ListDeviceProfileRequest{
		Limit:          Limit64,
		OrganizationId: orgId,
		ApplicationId:  appId,
	}); err == nil && resp.Result != nil {
		fmt.Println("profiles", resp.Result)
		for _, profile := range resp.Result {
			if profile.Name == name {
				return profile.Id, nil
			}
		}
	}

	//创建profile
	var resp1 *api.CreateDeviceProfileResponse
	if resp1, err = client.Create(ctx, &api.CreateDeviceProfileRequest{
		DeviceProfile: &api.DeviceProfile{
			NetworkServerId:      netId,
			OrganizationId:       orgId,
			Name:                 name,
			MacVersion:           "1.0.2",
			RegParamsRevision:    "A",
			MaxEirp:              20,
			PayloadCodec:         "CUSTOM_JS",
			PayloadDecoderScript: codec,
			UplinkInterval: &duration.Duration{
				Seconds: 600,
			},
			AdrAlgorithmId: "default",
		},
	}); err == nil {
		return resp1.Id, nil
	}

	return "", err
}

func DeleteProfile(conn *grpc.ClientConn, ctx context.Context, orgId int64, appId int64, name string) (err error) {
	client := api.NewDeviceProfileServiceClient(conn)
	var resp *api.ListDeviceProfileResponse
	if resp, err = client.List(ctx, &api.ListDeviceProfileRequest{
		Limit:          Limit64,
		OrganizationId: orgId,
		ApplicationId:  appId,
	}); err == nil && resp.Result != nil {
		for _, profile := range resp.Result {
			if profile.Name == name {
				if _, err = client.Delete(ctx, &api.DeleteDeviceProfileRequest{
					Id: profile.Id,
				}); err == nil {
					fmt.Println("profile delete success")
				}
			}
		}
	}
	return
}

func CreateGateway(conn *grpc.ClientConn, ctx context.Context, gateWayId string, name string, netId int64, orgId int64) (err error) {
	client := api.NewGatewayServiceClient(conn)

	var resp *api.GetGatewayResponse
	if resp, err = client.Get(ctx, &api.GetGatewayRequest{
		Id: gateWayId,
	}); err == nil && resp.Gateway != nil {
		fmt.Println("gateway is exist")
		return
	}

	if _, err = client.Create(ctx, &api.CreateGatewayRequest{
		Gateway: &api.Gateway{
			Id:              gateWayId,
			Name:            name,
			OrganizationId:  orgId,
			NetworkServerId: netId,
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
		Id: gateWayId,
	}); err == nil && resp.Gateway != nil {
		if _, err := client.Update(ctx, &api.UpdateGatewayRequest{
			Gateway: &api.Gateway{
				Id:   gateWayId,
				Name: name,
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
		Id: gateWayId,
	}); err == nil {
		fmt.Println("gateway delete success")
	} else {
		fmt.Println("gateway delete fail", err)
	}
	return
}

func CreateDevice(conn *grpc.ClientConn, ctx context.Context, DevEUI string, name string, deviceProfileId string, applicationId int64) (err error) {
	client := api.NewDeviceServiceClient(conn)
	if _, err = client.Create(ctx, &api.CreateDeviceRequest{
		Device: &api.Device{
			DevEui:          DevEUI,
			Name:            name,
			ApplicationId:   applicationId,
			DeviceProfileId: deviceProfileId,
			IsDisabled:      false,
			SkipFCntCheck:   true,
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
				IsDisabled:      false,
				SkipFCntCheck:   true,
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
