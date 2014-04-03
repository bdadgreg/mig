/* Kill a process using its PID

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

package pidkill

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type Parameters struct {
	Elements map[string][]int `json:"elements"`
}

func NewParameters() (p Parameters) {
	return
}

type Results struct {
	Success       bool                         `json:"success"`
	Elements      map[string]map[string]string `json:"results"`
	Error         string                       `json:"error,omitempty"`
}

func NewResults() *Results {
	return &Results{Elements: make(map[string]map[string]string),
		Success:       false}
}

func (p Parameters) Validate() (err error) {
	for name, pids := range p.Elements {
		for _, pid := range pids {
			if pid < 2 || pid > 65535 {
				return fmt.Errorf("In '%s', PID '%s' is not in the range [2:65535]", name, pid)
			}
		}
	}
	return
}

func Run(Args []byte) string {
	params := NewParameters()
	results := NewResults()

	err := json.Unmarshal(Args, &params.Elements)
	if err != nil {
		panic(err)
	}

	err = params.Validate()
	if err != nil {
		panic(err)
	}

	results.Success = true
	for name, pids := range params.Elements {
		for _, pid := range pids {
			pidstr := strconv.Itoa(pid)
			process, err := os.FindProcess(pid)
			if err != nil {
				results.Elements[name] = map[string]string{pidstr: fmt.Sprintf("%v", err)}
				results.Success = false
			} else {
				results.Elements[name] = map[string]string{pidstr: "found"}
				err = process.Kill()
				if err != nil {
					results.Elements[name] = map[string]string{pidstr: fmt.Sprintf("%v", err)}
					results.Success = false
				} else {
					results.Elements[name] = map[string]string{pidstr: "killed"}
				}
			}
		}
	}
	jsonOutput, err := json.Marshal(*results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}
