// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2019-2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package driver

const (
	LoraEUI      = "eui"
	LoraCodec    = "codec"
	LoraProtocol = "Lora"
	LoraGateway  = "Gateway"
)

// LoraProtocolParams holds end device protocol parameters
type LoraProtocolParams struct {
	EUI   string // 设备EUI、网关EUI
	Codec string // 设备编解码器
}
