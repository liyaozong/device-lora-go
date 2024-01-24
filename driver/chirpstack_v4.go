//go:build chirpstack4
// +build chirpstack4

package driver

import (
	"context"

	"github.com/edgexfoundry/device-lora-go/config"
	v4 "github.com/edgexfoundry/device-lora-go/utils/v4"
	"google.golang.org/grpc"
)

type ChirpStack struct {
	conn          *grpc.ClientConn
	config        config.ChirpStackConfig
	TenantId      string
	ApplicationId string
}

func (c *ChirpStack) Init() (err error) {
	c.conn, err = grpc.Dial(c.config.Host, grpc.WithInsecure())

	c.TenantId, c.ApplicationId, err = v4.Init(c.conn, c.config)
	return
}

func (c *ChirpStack) Login() (ctx context.Context, err error) {
	ctx, err = v4.Login(c.conn, c.config.Username, c.config.Password)
	return
}

func (c *ChirpStack) CreateProfile(ctx context.Context, name string, codec string) (id string, err error) {
	id, err = v4.CreateProfile(c.conn, ctx, c.TenantId, name, codec)
	return
}

func (c *ChirpStack) DeleteProfile(ctx context.Context, name string) (err error) {
	err = v4.DeleteProfile(c.conn, ctx, c.TenantId, name)
	return
}

func (c *ChirpStack) CreateGateway(ctx context.Context, gateWayId string, name string) (err error) {
	err = v4.CreateGateway(c.conn, ctx, gateWayId, name, c.TenantId)
	return
}

func (c *ChirpStack) UpdateGateway(ctx context.Context, gateWayId string, name string) (err error) {
	err = v4.UpdateGateway(c.conn, ctx, gateWayId, name)
	return
}

func (c *ChirpStack) DeleteGateway(ctx context.Context, gateWayId string) (err error) {
	err = v4.DeleteGateway(c.conn, ctx, gateWayId)
	return
}

func (c *ChirpStack) CreateDevice(ctx context.Context, DevEUI string, name string, deviceProfileId string) (err error) {
	err = v4.CreateDevice(c.conn, ctx, DevEUI, name, deviceProfileId, c.ApplicationId)
	return
}

func (c *ChirpStack) ActivateDevice(ctx context.Context, DevEUI string, key string) (err error) {
	err = v4.ActivateDevice(c.conn, ctx, DevEUI, key)
	return
}

func (c *ChirpStack) UpdateDevice(ctx context.Context, DevEUI string, name string) (err error) {
	err = v4.UpdateDevice(c.conn, ctx, DevEUI, name)
	return
}

func (c *ChirpStack) DeleteDevice(ctx context.Context, deviceName string, DevEUI string) (err error) {
	err = v4.DeleteDevice(c.conn, ctx, deviceName, DevEUI)
	return
}
