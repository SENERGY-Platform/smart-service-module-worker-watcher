/*
 * Copyright (c) 2026 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package worker

import (
	"encoding/json"
	"reflect"
	"testing"

	lib_model "github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
)

func TestCriteriaGetAspectIds(t *testing.T) {
	aid := "aid"
	other := "aid3"
	empty := ""

	t.Run("aspect ids are returned as listed", func(t *testing.T) {
		criteria := Criteria{AspectIds: []string{"aid1", "aid2"}}
		if !reflect.DeepEqual(criteria.GetAspectIds(), []string{"aid1", "aid2"}) {
			t.Error(criteria.GetAspectIds())
		}
	})

	t.Run("deprecated aspect id is an alias for a single element list", func(t *testing.T) {
		withAspectId := Criteria{AspectId: &aid}
		withAspectIds := Criteria{AspectIds: []string{"aid"}}
		if !reflect.DeepEqual(withAspectId.GetAspectIds(), withAspectIds.GetAspectIds()) {
			t.Error(withAspectId.GetAspectIds(), withAspectIds.GetAspectIds())
		}
	})

	t.Run("deprecated aspect id already in the list is not repeated", func(t *testing.T) {
		criteria := Criteria{AspectId: &aid, AspectIds: []string{"aid", "aid2"}}
		if !reflect.DeepEqual(criteria.GetAspectIds(), []string{"aid", "aid2"}) {
			t.Error(criteria.GetAspectIds())
		}
	})

	t.Run("deprecated aspect id not in the list is added", func(t *testing.T) {
		criteria := Criteria{AspectId: &other, AspectIds: []string{"aid1", "aid2"}}
		if !reflect.DeepEqual(criteria.GetAspectIds(), []string{"aid1", "aid2", "aid3"}) {
			t.Error(criteria.GetAspectIds())
		}
	})

	t.Run("adding the deprecated aspect id leaves the listed aspect ids untouched", func(t *testing.T) {
		aspectIds := make([]string, 2, 3)
		copy(aspectIds, []string{"aid1", "aid2"})
		criteria := Criteria{AspectId: &other, AspectIds: aspectIds}
		criteria.GetAspectIds()
		if !reflect.DeepEqual(aspectIds, []string{"aid1", "aid2"}) || aspectIds[:3][2] != "" {
			t.Error(aspectIds[:3])
		}
	})

	t.Run("empty deprecated aspect id is ignored", func(t *testing.T) {
		criteria := Criteria{AspectId: &empty}
		if len(criteria.GetAspectIds()) != 0 {
			t.Error(criteria.GetAspectIds())
		}
	})

	t.Run("no aspect is no aspect", func(t *testing.T) {
		criteria := Criteria{}
		if len(criteria.GetAspectIds()) != 0 {
			t.Error(criteria.GetAspectIds())
		}
	})
}

func TestCriteriaJson(t *testing.T) {
	t.Run("aspect ids are read from aspect_ids", func(t *testing.T) {
		criteria := []Criteria{}
		err := json.Unmarshal([]byte(`[{"function_id":"fid","aspect_ids":["aid1","aid2"]}]`), &criteria)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(criteria[0].GetAspectIds(), []string{"aid1", "aid2"}) {
			t.Error(criteria[0].GetAspectIds())
		}
	})

	t.Run("deprecated aspect_id is still read", func(t *testing.T) {
		criteria := []Criteria{}
		err := json.Unmarshal([]byte(`[{"function_id":"fid","aspect_id":"aid"}]`), &criteria)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(criteria[0].GetAspectIds(), []string{"aid"}) {
			t.Error(criteria[0].GetAspectIds())
		}
	})

	t.Run("empty aspect ids are not written", func(t *testing.T) {
		fid := "fid"
		b, err := json.Marshal(Criteria{FunctionId: &fid})
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != `{"interaction":null,"function_id":"fid","device_class_id":null,"aspect_id":null}` {
			t.Error(string(b))
		}
	})
}

func TestWatchDevicesByCriteriaWithAspectIds(t *testing.T) {
	cases := []struct {
		name     string
		varName  string
		endpoint string
		get      func(w *Worker, task lib_model.CamundaExternalTask) (model.HttpRequest, error)
	}{
		{
			name:     "devices",
			varName:  "watcher.watch_devices_by_criteria",
			endpoint: "http://device-selection-url:8080/v2/query/selectables?include_devices=true",
			get:      (*Worker).getWatchedDevicesHttpRequest,
		},
		{
			name:     "modified devices",
			varName:  "watcher.watch_modified_devices_by_criteria",
			endpoint: "http://device-selection-url:8080/v2/query/selectables?include_devices=true&include_id_modified=true",
			get:      (*Worker).getWatchedModifiedDevicesHttpRequest,
		},
	}
	w := &Worker{config: configuration.Config{
		WorkerParamPrefix:  "watcher.",
		DeviceSelectionUrl: "http://device-selection-url:8080",
	}}
	taskWith := func(varName string, value string) lib_model.CamundaExternalTask {
		return lib_model.CamundaExternalTask{Variables: map[string]lib_model.CamundaVariable{varName: {Value: value}}}
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Run("forwards aspect ids to the device-selection unchanged", func(t *testing.T) {
				body := `[{"function_id":"fid","aspect_ids":["aid1","aid2"]}]`
				req, err := c.get(w, taskWith(c.varName, body))
				if err != nil {
					t.Fatal(err)
				}
				if string(req.Body) != body {
					t.Error(string(req.Body))
				}
				if req.Endpoint != c.endpoint || req.Method != "POST" || !req.AddAuthToken {
					t.Errorf("%#v", req)
				}
			})

			t.Run("forwards the deprecated aspect id unchanged", func(t *testing.T) {
				body := `[{"function_id":"fid","aspect_id":"aid"}]`
				req, err := c.get(w, taskWith(c.varName, body))
				if err != nil {
					t.Fatal(err)
				}
				if string(req.Body) != body {
					t.Error(string(req.Body))
				}
			})

			t.Run("forwards aspect ids next to the deprecated aspect id unchanged", func(t *testing.T) {
				body := `[{"function_id":"fid","aspect_id":"aid","aspect_ids":["aid1"]}]`
				req, err := c.get(w, taskWith(c.varName, body))
				if err != nil {
					t.Fatal(err)
				}
				if string(req.Body) != body {
					t.Error(string(req.Body))
				}
			})

			t.Run("rejects aspect ids that are not a list", func(t *testing.T) {
				_, err := c.get(w, taskWith(c.varName, `[{"function_id":"fid","aspect_ids":"aid"}]`))
				if err == nil {
					t.Error("expected error")
				}
			})
		})
	}
}
