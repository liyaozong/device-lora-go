//go:build chirpstack3
// +build chirpstack3

package driver

import (
	"context"

	"github.com/edgexfoundry/device-lora-go/config"
	v3 "github.com/edgexfoundry/device-lora-go/utils/v3"
	"google.golang.org/grpc"
)

var (
	Limit64 int64 = 100
)

type ChirpStack struct {
	conn            *grpc.ClientConn
	config          config.ChirpStackConfig
	NetWorkServerId int64
	OrganizationId  int64
	ApplicationId   int64
}

func (c *ChirpStack) Init() (err error) {
	c.conn, err = grpc.Dial(c.config.Host, grpc.WithInsecure())

	c.NetWorkServerId, c.OrganizationId, c.ApplicationId, err = v3.Init(c.conn, c.config)
	return
}

func (c *ChirpStack) Login() (ctx context.Context, err error) {
	ctx, err = v3.Login(c.conn, c.config.Username, c.config.Password)
	return
}

func (c *ChirpStack) CreateProfile(ctx context.Context, name string, codec string) (id string, err error) {
	id, err = v3.CreateProfile(c.conn, ctx, c.NetWorkServerId, c.OrganizationId, c.ApplicationId, name, codec)
	return
}

func (c *ChirpStack) DeleteProfile(ctx context.Context, name string) (err error) {
	err = v3.DeleteProfile(c.conn, ctx, c.OrganizationId, c.ApplicationId, name)
	return
}

func (c *ChirpStack) CreateGateway(ctx context.Context, gateWayId string, name string) (err error) {
	err = v3.CreateGateway(c.conn, ctx, gateWayId, name, c.OrganizationId, c.NetWorkServerId)
	return
}

func (c *ChirpStack) UpdateGateway(ctx context.Context, gateWayId string, name string) (err error) {
	err = v3.UpdateGateway(c.conn, ctx, gateWayId, name)
	return
}

func (c *ChirpStack) DeleteGateway(ctx context.Context, gateWayId string) (err error) {
	err = v3.DeleteGateway(c.conn, ctx, gateWayId)
	return
}

func (c *ChirpStack) CreateDevice(ctx context.Context, DevEUI string, name string, deviceProfileId string) (err error) {
	err = v3.CreateDevice(c.conn, ctx, DevEUI, name, deviceProfileId, c.ApplicationId)
	return
}

func (c *ChirpStack) ActivateDevice(ctx context.Context, DevEUI string, key string) (err error) {
	err = v3.ActivateDevice(c.conn, ctx, DevEUI, key)
	return
}

func (c *ChirpStack) UpdateDevice(ctx context.Context, DevEUI string, name string) (err error) {
	err = v3.UpdateDevice(c.conn, ctx, DevEUI, name)
	return
}

func (c *ChirpStack) DeleteDevice(ctx context.Context, deviceName string, DevEUI string) (err error) {
	err = v3.DeleteDevice(c.conn, ctx, deviceName, DevEUI)
	return
}
