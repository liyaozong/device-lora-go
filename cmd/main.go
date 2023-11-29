//

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

package main

import (
	"github.com/edgexfoundry/device-sdk-go/v3/pkg/startup"

	device_lora "github.com/edgexfoundry/device-lora-go"
	"github.com/edgexfoundry/device-lora-go/driver"
)

const (
	serviceName string = "device-lora"
)

func main() {
	sd := driver.LoraDriver{}
	startup.Bootstrap(serviceName, device_lora.Version, &sd)
}
