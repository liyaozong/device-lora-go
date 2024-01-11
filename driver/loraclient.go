package driver

import (
	"errors"
	"fmt"

	"github.com/edgexfoundry/go-mod-core-contracts/v3/models"
)

func (driver *LoraDriver) AddLoraDevice(chirp *ChirpStack, device models.Device, profile models.DeviceProfile, protocolParams LoraProtocolParams) (err error) {
	// 登录chirpstack
	ctx := chirp.Login()

	var profileId string
	// lorawan返回的是json对象数据，
	if len(profile.DeviceResources) == 1 {
		optional := profile.DeviceResources[0].Properties.Optional
		if _, ok := optional[CODEC]; !ok {
			return errors.New("optional codec not exists")
		}
		codec := fmt.Sprintf("%v", optional[CODEC])

		profileId, err = chirp.CreateProfile(ctx, profile.Name, codec)
	}

	if protocolParams.Gateway {
		// 创建网关
		err = chirp.CreateGateway(ctx, protocolParams.EUI, device.Name)
	} else {
		// 创建设备
		if err = chirp.CreateDevice(ctx, protocolParams.EUI, device.Name, profileId); err == nil {
			// 激活设备
			err = chirp.ActivateDevice(ctx, protocolParams.EUI, chirp.config.ActivateKey)

			// 添加监听
			listener := Listener{
				driver:     driver,
				DeviceName: device.Name,
				config:     chirp.config,
				Stop:       false,
			}
			driver.listeners[device.Name] = listener
			go listener.Listening(chirp, ctx, protocolParams.EUI)
		}
	}
	return
}

func (driver *LoraDriver) UpdateLoraDevice(chirp *ChirpStack, device models.Device, protocolParams LoraProtocolParams) (err error) {
	// 登录chirpstack
	ctx := chirp.Login()
	if protocolParams.Gateway {
		// 更新网关
		err = chirp.UpdateGateway(ctx, protocolParams.EUI, device.Name)
	} else {
		// 更新设备
		if err = chirp.UpdateDevice(ctx, protocolParams.EUI, device.Name); err == nil {
			// 更新监听，主要是EUI的变化，这里检测不到EUI的变化，简单处理
			if listener, ok := driver.listeners[device.Name]; ok {
				// 删除旧的监听
				listener.Cancel()
				delete(driver.listeners, device.Name)

				// 添加新的监听
				listener := Listener{
					driver:     driver,
					DeviceName: device.Name,
					config:     chirp.config,
					Stop:       false,
				}
				driver.listeners[device.Name] = listener
				go listener.Listening(chirp, ctx, protocolParams.EUI)
			}
		}
	}

	return
}

func (driver *LoraDriver) RemoveLoraDevice(chirp *ChirpStack, deviceName string, protocolParams LoraProtocolParams) (err error) {
	// 登录chirpstack
	ctx := chirp.Login()
	if protocolParams.Gateway {
		// 删除网关
		err = chirp.DeleteGateway(ctx, protocolParams.EUI)
	} else {
		// 删除设备
		if err = chirp.DeleteDevice(ctx, deviceName, protocolParams.EUI); err == nil {
			// 删除监听
			if listener, ok := driver.listeners[deviceName]; ok {
				listener.Cancel()
				delete(driver.listeners, deviceName)
			}
		}
	}

	// 删除设备profile，codec转移到profile里面后由于取不到profileName，所以无法删除profile
	// driver.deleteProfile(ctx, driver.tenantId, profileName)

	return
}
