
# 部署Chirpstack

用docker compose部署chirpstack是最方便捷的，首先克隆chirpstack-docker库

https://github.com/chirpstack/chirpstack-docker.git

国内使用cn470_0中国区的LoraWan网络配置

### 在configuration/chirpstack/chirpstack.toml文件中添加频点支持：
+    "cn470_0",
+    "cn470_1",
+    "cn470_2",
+    "cn470_3",
+    "cn470_4",
+    "cn470_5",
+    "cn470_6",
+    "cn470_7",
+    "cn470_8",
+    "cn470_9",
+    "cn470_10",
+    "cn470_11",

#### 用openssl rand -base64 32生成随机密钥，并填入secret字段
-  secret="you-must-replace-this"
+  secret="q65ldYg6AIj6iM/RzGxBqNQm0m016SIEq/DGn8tNzrU="

### 修改docker-compose.yml文件

删除欧洲区配置
-      - INTEGRATION__MQTT__EVENT_TOPIC_TEMPLATE=eu868/gateway/{{ .GatewayID }}/event/{{ .EventType }}
-      - INTEGRATION__MQTT__STATE_TOPIC_TEMPLATE=eu868/gateway/{{ .GatewayID }}/state/{{ .StateType }}
-      - INTEGRATION__MQTT__COMMAND_TOPIC_TEMPLATE=eu868/gateway/{{ .GatewayID }}/command/#

使用中国区配置
+      - INTEGRATION__MQTT__EVENT_TOPIC_TEMPLATE=cn470_0/gateway/{{ .GatewayID }}/event/{{ .EventType }}
+      - INTEGRATION__MQTT__STATE_TOPIC_TEMPLATE=cn470_0/gateway/{{ .GatewayID }}/state/{{ .StateType }}
+      - INTEGRATION__MQTT__COMMAND_TOPIC_TEMPLATE=cn470_0/gateway/{{ .GatewayID }}/command/#


#### 修改chirpstack-gateway-bridge-basicstation配置

删除欧洲区配置
-    command: -c /etc/chirpstack-gateway-bridge/chirpstack-gateway-bridge-basicstation-eu868.toml

使用中国区配置
+    command: -c /etc/chirpstack-gateway-bridge/chirpstack-gateway-bridge-basicstation-cn470_0.toml


# LoraWan网关

有些LoraWan网关自带chirpstack功能，也可以直接使用，如下RAK-7243（深圳瑞科慧联）文档说明：

https://docs.rakwireless.com.cn/Product-Categories/WisGate/RAK7243/Quickstart/


# LoraWan设备

这里介绍CC10LD REV.B (广州欧创智能)型号LoraWan产品的使用，这款设备可接入RS485协议的各种传感器，其入网配置步骤如下：

将设备通过usb直连到PC个人电脑，使用厂商提供的QSerialTool_v1.14.180111专用工具，或者sscom等串口工具进行配置，支持AT指定配置。

* +++\r                            （进入at命令模式）


基本配置
* at+mode=mac\r                     (配置为lorawan通信模式)
* at+ch=0,486300000,0,5\r           (配置1通道频率，与网关上行1通道一致)
* at+ch=1,486500000,0,5\r           (配置2通道频率，与网关上行2通道一致)
* at+ch=2,486700000,0,5\r           (配置3通道频率，与网关上行3通道一致)
* at+ch=3,486900000,0,5\r           (配置4通道频率，与网关上行4通道一致)
* at+ch=4,487100000,0,5\r           (配置5通道频率，与网关上行5通道一致)
* at+ch=5,487300000,0,5\r           (配置6通道频率，与网关上行6通道一致)
* at+ch=6,487500000,0,5\r           (配置7通道频率，与网关上行7通道一致)
* at+ch=7,487700000,0,5\r           (配置8通道频率，与网关上行8通道一致)
* at+join=abp\r                    （配置终端入网方式为abp）


入网配置，谨记：入网配置完成后，需重启设备生效
* at+appskey=abp,bc67cd6eb45a08d975050b1887b93c23\r (配置终端应用密钥)
* at+nwkskey=abp,bc67cd6eb45a08d975050b1887b93c23\r (配置终端网络会话密钥)
* at+devaddr=0x6d3d77\r             (配置设备网络短地址，chirpstack设备详情里找到)


重启后，可以用如下命令查询是否入网
* at+join=?\r                      （返回带有joined说明入网成功）


配置RS485相关参数，该设备中有自动轮询器，可周期读取串口数据
* at+pkthdr=on\r                   （自动轮询开，轮询读取串口数据）
* at+rx2=0,505300000\r             （配置rx2）
* at+pptm=add,5,poll,010301f400028405\r  （添加传感器命令，add后为命令编号，上报数据时会在数据开头增加一个字节的编号数据，poll后为串口读取命令）
* at+pptm=?\r                      （查询所有传感器命令）
* at+pptm=del,5\r                  （删除传感器命令）
* at+pptmcfg=10,5,5\r              （设置轮询时间）
* at+pptmcfg=?\r                   （查询轮询时间配置）

LoraWan设备入网后，也可通过chirpstack提供的通道进行远程命令下发
该设备下发固定端口为：222
如：at+pptm=add,1,poll,0000000(不需要\r)，经base64编码为：YXQrcHB0bT1hZGQsMSxwb2xsLDAwMDAwMDA= 可直接在web上操作下发，也可以通过Chirpstack Api下发：

`/api/devices/{dev_eui}/queue  (POST)`
> 请求json结构体如下：
`
{
    "deviceQueueItem": {
        "fPort":222,
        "data":"YXQrcHB0bT1hZGQsMSxwb2xsLDAwMDAwMDA=","devEUI":"00010111ff001293"
    }
}
`


配置完成，可以用如下命令查看配置参数
* at+parm=?\r                      （查看配置参数）
* at+join=?\r                      （查询入网NS）


# 运行device-lora-go

cd cmd
go run -tags=chirpstack3 main.go --cp=consul://edgex-core-consul:8500 --registry
go run -tags=chirpstack4 main.go --cp=consul://edgex-core-consul:8500 --registry