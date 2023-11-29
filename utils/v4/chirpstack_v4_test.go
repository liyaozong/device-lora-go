package v4

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/chirpstack/chirpstack/api/go/v4/api"
	"github.com/chirpstack/chirpstack/api/go/v4/common"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// configuration
var (
	Limit32     uint32 = 100
	Key         string = "bc67cd6eb45a08d975050b1887b93c23"
	Host        string = "172.16.64.157:8082"
	Admin       string = "admin"
	Password    string = "admin"
	tenantId           = "52f14cd4-c6f1-4fbd-8f87-4025e1d49242"
	globalToken        = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJjaGlycHN0YWNrIiwiaXNzIjoiY2hpcnBzdGFjayIsInN1YiI6Ijg4MzY5ZDA4LWFjODItNGJmNy1iYjhmLWY4MjdkMjAxZWYxZiIsInR5cCI6ImtleSJ9.4fOyiEeLsJw8T0CosXkpUR7KwpJG6XX8xNeImL2r4G8"
	tenantToken        = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJjaGlycHN0YWNrIiwiaXNzIjoiY2hpcnBzdGFjayIsInN1YiI6ImU4MTg2MDA4LTQ2MGItNGVlNi04NTczLTk3MzllOGI4ZDBlYiIsInR5cCI6ImtleSJ9.QVc7rzjzji1b28oBVZUnN-aWIWol_LbJtx9syLOgu38"
)

type APIToken string

func (a APIToken) GetRequestMetadata(ctx context.Context, url ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", a),
	}, nil
}

func (a APIToken) RequireTransportSecurity() bool {
	return false
}

func dial(token string) (conn *grpc.ClientConn, err error) {
	// define gRPC dial options
	dialOpts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(APIToken(token)),
		grpc.WithInsecure(), // remove this when using TLS
	}

	// connect to the gRPC server
	conn, err = grpc.Dial(Host, dialOpts...)
	return
}

func TestLogin(t *testing.T) {
	conn, err := grpc.Dial(Host, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	client := api.NewInternalServiceClient(conn)
	// 调用Login方法  参数一为context上下文、二为LoginRequest结构体
	if resp, err := client.Login(context.Background(), &api.LoginRequest{
		Email:    Admin,
		Password: Password,
	}); err == nil {
		fmt.Println("jwt", resp.Jwt)
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+resp.Jwt))

		client2 := api.NewInternalServiceClient(conn)
		if resp, err := client2.ListApiKeys(ctx, &api.ListApiKeysRequest{
			Limit: Limit32,
			// IsAdmin: true,
			TenantId: tenantId,
		}); err != nil {
			fmt.Println("keys", err)
		} else {
			fmt.Println("keys", resp.Result)
		}

		DevEUI := "eb1e9244e686bd11"
		DevName := "Test device 2"

		client := api.NewDeviceServiceClient(conn)
		if resp, err := client.Get(ctx, &api.GetDeviceRequest{
			DevEui: DevEUI,
		}); err == nil {
			fmt.Println("dev profile id", resp.Device.DeviceProfileId)
		}
		if _, err := client.Update(ctx, &api.UpdateDeviceRequest{
			Device: &api.Device{
				DevEui:          DevEUI,
				Name:            DevName,
				ApplicationId:   "56687cd8-38fe-4e9f-b30f-7149b28414e9",
				DeviceProfileId: "74f9882d-77c6-4ac3-9fb7-54b4ce4b5447",
				SkipFcntCheck:   true,
			},
		}); err != nil {
			fmt.Println("dev create fail", err)
		}

		client3 := api.NewDeviceProfileServiceClient(conn)
		resp2, err := client3.List(ctx, &api.ListDeviceProfilesRequest{
			Limit:    Limit32,
			TenantId: tenantId,
			Search:   "Lora Device Profile",
		})
		if err != nil {
			fmt.Println("dev profile", err)
		} else {
			fmt.Println("dev profile", resp2.Result)
		}
	}
}

func TestChirp(t *testing.T) {
	conn, _ := dial(globalToken)
	client := api.NewTenantServiceClient(conn)
	if resp, err := client.List(context.Background(), &api.ListTenantsRequest{
		Limit: Limit32,
	}); err != nil {
		fmt.Println("tenants", err)
	} else {
		fmt.Println("tenants", resp.Result)
	}
}

func TestTenant(t *testing.T) {
	name := "application1"

	// define gRPC dial options
	conn, _ := dial(tenantToken)
	client := api.NewApplicationServiceClient(conn)

	client.Create(context.Background(), &api.CreateApplicationRequest{
		Application: &api.Application{
			Id:       uuid.NewV4().String(),
			Name:     name,
			TenantId: tenantId,
		},
	})
	resp, err := client.List(context.Background(), &api.ListApplicationsRequest{
		Limit:    Limit32,
		TenantId: tenantId,
	})
	if err != nil {
		fmt.Println("apps", err)
	} else {
		fmt.Println("apps", resp.Result)
	}
}

func TestDevice(t *testing.T) {
	DevEUI := "9e13b5893728d5f6"
	DevName := "Test device 2"

	appId := "56687cd8-38fe-4e9f-b30f-7149b28414e9"
	devProfileId := "74f9882d-77c6-4ac3-9fb7-54b4ce4b5447"

	key := "bc67cd6eb45a08d975050b1887b93c23"

	conn, _ := dial(tenantToken)

	client := api.NewDeviceServiceClient(conn)
	if _, err := client.Create(context.Background(), &api.CreateDeviceRequest{
		Device: &api.Device{
			DevEui:          DevEUI,
			Name:            DevName,
			ApplicationId:   appId,
			DeviceProfileId: devProfileId,
			SkipFcntCheck:   true,
		},
	}); err != nil {
		fmt.Println("dev create fail", err)
	} else {
		fmt.Println("dev create success")
		// if _, err := client.CreateKeys(context.Background(), &api.CreateDeviceKeysRequest{
		// 	DeviceKeys: &api.DeviceKeys{
		// 		DevEui: DevEUI,
		// 		NwkKey: key,
		// 		AppKey: key,
		// 	},
		// }); err != nil {
		// 	fmt.Println("dev createKeys fail", err)
		// } else {
		// 	if resp, err := client.GetKeys(context.Background(), &api.GetDeviceKeysRequest{
		// 		DevEui: DevEUI,
		// 	}); err == nil {
		// 		fmt.Println("dev createKeys", resp.DeviceKeys)
		// 	}
		// }

		if resp, err := client.GetRandomDevAddr(context.Background(), &api.GetRandomDevAddrRequest{
			DevEui: DevEUI,
		}); err != nil {
			fmt.Println("dev get addr fail", err)
		} else {
			fmt.Println("dev get addr", resp.DevAddr)
			if _, err := client.Activate(context.Background(), &api.ActivateDeviceRequest{
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
	}

	if resp4, err := client.List(context.Background(), &api.ListDevicesRequest{
		Limit:         Limit32,
		ApplicationId: appId,
	}); err != nil {
		fmt.Println("devs", err)
	} else {
		fmt.Println("devs", resp4.Result)
	}
}

func TestUpdateDevice(t *testing.T) {
	DevEUI := "eb1e9244e686bd11"
	DevName := "Test device 2"

	conn, _ := dial(tenantToken)

	client := api.NewDeviceServiceClient(conn)
	if _, err := client.Update(context.Background(), &api.UpdateDeviceRequest{
		Device: &api.Device{
			DevEui:        DevEUI,
			Name:          DevName,
			SkipFcntCheck: true,
		},
	}); err != nil {
		fmt.Println("dev create fail", err)
	}
}

func TestDeviceEvent(t *testing.T) {
	DevEUI := "9d13b5893728d5f6"

	conn, _ := dial(globalToken)
	client := api.NewInternalServiceClient(conn)
	if stream, err := client.StreamDeviceEvents(context.Background(), &api.StreamDeviceEventsRequest{
		DevEui: DevEUI,
	}); err != nil {
		fmt.Println("dev get addr fail", err)
	} else {
		// go func() {
		for {
			resp, err := stream.Recv()
			if err != nil {
				fmt.Println("dev stream fail", err)
				return
			}

			var payloadJson map[string]any
			err = json.Unmarshal([]byte(resp.Body), &payloadJson)
			if err != nil {
				continue
			}
			fmt.Println("dev stream event", payloadJson)
		}
		// }()
	}
}

func TestGateway(t *testing.T) {
	gateWayId := "a84caecb2846bbf8"
	name := "Test gatway 1"
	var statsInterval uint32 = 30

	conn, _ := dial(tenantToken)
	client := api.NewGatewayServiceClient(conn)
	if _, err := client.Create(context.Background(), &api.CreateGatewayRequest{
		Gateway: &api.Gateway{
			GatewayId:     gateWayId,
			Name:          name,
			TenantId:      tenantId,
			StatsInterval: statsInterval,
		},
	}); err != nil {
		fmt.Println("gateway create fail", err)
	} else {
		fmt.Println("gateway create success")
	}

	resp3, err := client.List(context.Background(), &api.ListGatewaysRequest{
		Limit:    Limit32,
		TenantId: tenantId,
	})
	if err != nil {
		fmt.Println("gateway", err)
	} else {
		fmt.Println("gateway", resp3.Result)
	}
}

func TestDevProfile(t *testing.T) {
	name := "Lora Device Profile"
	var uplinkInterval uint32 = 3600
	adrAlgorithm := "default" // options: default, lr_fhss, lora_lr_fhss

	conn, _ := dial(tenantToken)
	client := api.NewDeviceProfileServiceClient(conn)
	client.Create(context.Background(), &api.CreateDeviceProfileRequest{
		DeviceProfile: &api.DeviceProfile{
			Id:                uuid.NewV4().String(),
			TenantId:          tenantId,
			Name:              name,
			Region:            common.Region_CN470,
			MacVersion:        common.MacVersion_LORAWAN_1_0_3,
			RegParamsRevision: common.RegParamsRevision_A,
			AdrAlgorithmId:    adrAlgorithm, // options: default, lr_fhss, lora_lr_fhss
			UplinkInterval:    uplinkInterval,
			// SupportsOtaa:      true,
		},
	})

	resp2, err := client.List(context.Background(), &api.ListDeviceProfilesRequest{
		Limit:    Limit32,
		TenantId: tenantId,
		Search:   "Lora Device Profile",
	})
	if err != nil {
		fmt.Println("dev profile", err)
	} else {
		fmt.Println("dev profile", resp2.Result)
	}
}
