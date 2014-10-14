// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Unmarshal's a Containers description json file. The json file contains
// an array of ContainerHint structs, each with a container's id and networkInterface
// This allows collecting stats about network interfaces configured outside docker
// and lxc
package raw

import (
	"flag"
	"encoding/json"
	"io/ioutil"
	"github.com/golang/glog"
)
var argContainerHints = flag.String("container_hints", "/etc/cadvisor/container_description.json", "container hints file")
type containerHints struct {
	All_hosts []containerHint `json:"all_hosts,omitempty"`
}

type containerHint struct {
	Id                string `json:"id,omitempty"`
	NetworkInterface *networkInterface `json:"network_interface,omitempty"`
}

type networkInterface struct {
	VethHost  string `json:"VethHost,omitempty"`
	VethChild string `json:"VethChild,omitempty"`
	NsPath    string `json:"NsPath,omitempty"`
}

func Unmarshal(containerHintsFile string) (containerHints, error) {
	dat, err := ioutil.ReadFile(containerHintsFile)
	glog.Infof("Read file: %s", string(dat))
	var cDesc containerHints
	if err == nil {
		err = json.Unmarshal(dat, &cDesc)
	}
	glog.Infof("Read json: %s", cDesc)
	return cDesc, err
}
