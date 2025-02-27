// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package instances

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	apperr "github.com/google/cloud-android-orchestration/pkg/app/errors"

	"github.com/google/go-cmp/cmp"
	"github.com/sergi/go-diff/diffmatchpatch"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

var testConfig = Config{
	GCP: &GCPIMConfig{
		ProjectID:       "google.com:test-project",
		HostImageFamily: "projects/test-project-releases/global/images/family/foo",
	},
}

var testNameGenerator = &testConstantNameGenerator{name: "foo"}

const fakeUsername = "johndoe"

type TestUser struct{}

func (i *TestUser) Username() string {
	return fakeUsername
}

func TestCreateHostInvalidRequests(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replyJSON(w, &compute.Operation{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := NewGCEInstanceManager(testConfig, testService, testNameGenerator)
	var validRequest = func() *apiv1.CreateHostRequest {
		return &apiv1.CreateHostRequest{
			HostInstance: &apiv1.HostInstance{
				GCP: &apiv1.GCPInstance{
					MachineType:    "n1-standard-1",
					MinCPUPlatform: "Intel Haswell",
				},
			},
		}
	}
	// Make sure the valid request is indeed valid.
	_, err := im.CreateHost("us-central1-a", validRequest(), &TestUser{})
	if err != nil {
		t.Fatalf("the valid request is not valid with error %+v", err)
	}
	var tests = []struct {
		corruptRequest func(r *apiv1.CreateHostRequest)
	}{
		{func(r *apiv1.CreateHostRequest) { r.HostInstance = nil }},
		{func(r *apiv1.CreateHostRequest) { r.HostInstance.Name = "foo" }},
		{func(r *apiv1.CreateHostRequest) { r.HostInstance.BootDiskSizeGB = 1 }},
		{func(r *apiv1.CreateHostRequest) { r.HostInstance.GCP = nil }},
		{func(r *apiv1.CreateHostRequest) { r.HostInstance.GCP.MachineType = "" }},
	}

	for _, test := range tests {
		req := validRequest()
		test.corruptRequest(req)
		_, err := im.CreateHost("us-central1-a", req, &TestUser{})
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) {
			t.Errorf("unexpected error <<\"%v\">>, want \"%T\"", err, appErr)
		}
	}
}

func TestCreateHostRequestPath(t *testing.T) {
	var pathSent string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathSent = r.URL.Path
		replyJSON(w, &compute.Operation{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := NewGCEInstanceManager(testConfig, testService, testNameGenerator)

	im.CreateHost("us-central1-a",
		&apiv1.CreateHostRequest{
			HostInstance: &apiv1.HostInstance{
				GCP: &apiv1.GCPInstance{
					MachineType:    "n1-standard-1",
					MinCPUPlatform: "Intel Haswell",
				},
			},
		},
		&TestUser{})

	expected := "/projects/google.com:test-project/zones/us-central1-a/instances"
	if pathSent != expected {
		t.Errorf("unexpected url path <<%s>>, want: %s", pathSent, expected)
	}
}

func TestCreateHostRequestBody(t *testing.T) {
	var bodySent compute.Instance
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(body, &bodySent)
		replyJSON(w, &compute.Operation{Name: "operation-1"})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := NewGCEInstanceManager(testConfig, testService, testNameGenerator)

	im.CreateHost("us-central1-a",
		&apiv1.CreateHostRequest{
			HostInstance: &apiv1.HostInstance{
				GCP: &apiv1.GCPInstance{
					MachineType:    "n1-standard-1",
					MinCPUPlatform: "Intel Haswell",
				},
			},
		},
		&TestUser{})

	expected := `{
  "disks": [
    {
      "boot": true,
      "initializeParams": {
        "sourceImage": "projects/test-project-releases/global/images/family/foo"
      }
    }
  ],
  "labels": {
    "cf-created_by": "` + fakeUsername + `"
  },
  "machineType": "zones/us-central1-a/machineTypes/n1-standard-1",
  "minCpuPlatform": "Intel Haswell",
  "name": "foo",
  "networkInterfaces": [
    {
      "accessConfigs": [
        {
          "name": "External NAT",
          "type": "ONE_TO_ONE_NAT"
        }
      ],
      "name": "projects/google.com:test-project/global/networks/default"
    }
  ]
}`
	r := prettyJSON(t, &bodySent)
	if r != expected {
		t.Errorf("unexpected body, diff: %s", diffPrettyText(r, expected))
	}
}

func TestCreateHostAcloudCompatible(t *testing.T) {
	var postedInstance compute.Instance
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(body, &postedInstance)
		replyJSON(w, &compute.Operation{Name: "operation-1"})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := GCEInstanceManager{
		Config: Config{
			GCP: &GCPIMConfig{
				ProjectID:        "google.com:test-project",
				HostImageFamily:  "projects/test-project-releases/global/images/family/foo",
				AcloudCompatible: true,
			},
		},
		Service:               testService,
		InstanceNameGenerator: testNameGenerator,
	}

	_, err := im.CreateHost("us-central1-a",
		&apiv1.CreateHostRequest{
			HostInstance: &apiv1.HostInstance{
				GCP: &apiv1.GCPInstance{
					MachineType:    "n1-standard-1",
					MinCPUPlatform: "Intel Haswell",
				},
			},
		},
		&TestUser{})

	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(fakeUsername, postedInstance.Labels[labelAcloudCreatedBy]); diff != "" {
		t.Errorf("created by acloud label (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("startup-script", postedInstance.Metadata.Items[0].Key); diff != "" {
		t.Errorf("metadata item key (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(acloudSetupScript, *postedInstance.Metadata.Items[0].Value); diff != "" {
		t.Errorf("metadata item value (-want +got):\n%s", diff)
	}
}

func TestCreateHostSuccess(t *testing.T) {
	expectedName := "operation-1"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o := &compute.Operation{
			Name:   expectedName,
			Status: "PENDING",
		}
		replyJSON(w, o)
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := NewGCEInstanceManager(testConfig, testService, testNameGenerator)

	op, _ := im.CreateHost("us-central1-a",
		&apiv1.CreateHostRequest{
			HostInstance: &apiv1.HostInstance{
				GCP: &apiv1.GCPInstance{
					MachineType:    "n1-standard-1",
					MinCPUPlatform: "Intel Haswell",
				},
			},
		},
		&TestUser{})

	if op.Name != expectedName {
		t.Errorf("expected <<%q>>, got: %q", expectedName, op.Name)
	}
	if op.Done {
		t.Error("expected not done.")
	}
}

func TestGetHostAddrRequestPath(t *testing.T) {
	var pathSent string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathSent = r.URL.Path
		replyJSON(w, &compute.Instance{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := NewGCEInstanceManager(testConfig, testService, testNameGenerator)

	im.GetHostAddr("us-central1-a", "foo")

	expected := "/projects/google.com:test-project/zones/us-central1-a/instances/foo"
	if pathSent != expected {
		t.Errorf("unexpected url path <<%q>>, want: %q", pathSent, expected)
	}
}

func TestGetHostAddrMissingNetworkInterface(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replyJSON(w, &compute.Instance{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := NewGCEInstanceManager(testConfig, testService, testNameGenerator)

	_, err := im.GetHostAddr("us-central1-a", "foo")

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("unexpected error <<\"%v\">>, want \"%T\"", err, appErr)
	}
}

func TestGetHostAddrSuccess(t *testing.T) {
	expectedIP := "10.128.0.63"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := &compute.Instance{
			NetworkInterfaces: []*compute.NetworkInterface{
				{
					NetworkIP: expectedIP,
				},
			},
		}
		replyJSON(w, i)
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := NewGCEInstanceManager(testConfig, testService, testNameGenerator)

	addr, _ := im.GetHostAddr("us-central1-a", "foo")

	if addr != expectedIP {
		t.Errorf("unexpected host address <<%q>>, want: %q", addr, expectedIP)
	}
}

func TestListHostsRequestQuery(t *testing.T) {
	var usedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usedQuery = r.URL.Query().Encode()
		replyJSON(w, &compute.InstanceList{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := NewGCEInstanceManager(testConfig, testService, testNameGenerator)
	req := &ListHostsRequest{
		MaxResults: 100,
		PageToken:  "foo",
	}

	im.ListHosts("us-central1-a", &TestUser{}, req)

	m, _ := url.ParseQuery(usedQuery)
	got, expected := m["filter"][0], "labels.cf-created_by:johndoe AND status=RUNNING"
	if got != expected {
		t.Errorf("expected <<%q>>, got %q", expected, got)
	}
	got, expected = m["maxResults"][0], "100"
	if got != expected {
		t.Errorf("expected <<%q>>, got %q", expected, got)
	}
}

func TestListHostsOverMaxResultsLimit(t *testing.T) {
	var usedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usedQuery = r.URL.Query().Encode()
		replyJSON(w, &compute.InstanceList{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := NewGCEInstanceManager(testConfig, testService, testNameGenerator)
	req := &ListHostsRequest{
		MaxResults: 501,
		PageToken:  "foo",
	}

	im.ListHosts("us-central1-a", &TestUser{}, req)

	m, _ := url.ParseQuery(usedQuery)
	got, expected := m["maxResults"][0], "500"
	if got != expected {
		t.Errorf("expected <<%q>>, got %q", expected, got)
	}
}

func TestListHostsSucceeds(t *testing.T) {
	i1 := &compute.Instance{
		Disks:          []*compute.AttachedDisk{{DiskSizeGb: 10}},
		Name:           "foo",
		MachineType:    "mt",
		MinCpuPlatform: "mcp",
	}
	i2 := &compute.Instance{
		Disks:          []*compute.AttachedDisk{{DiskSizeGb: 20}},
		Name:           "bar",
		MachineType:    "mtbaz",
		MinCpuPlatform: "mcpbaz",
	}
	nextPageToken := "test-token"
	items := []*compute.Instance{i1, i2}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		list := &compute.InstanceList{
			Items:         items,
			NextPageToken: "test-token",
		}
		replyJSON(w, list)
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := NewGCEInstanceManager(testConfig, testService, testNameGenerator)

	resp, err := im.ListHosts("us-central1-a", &TestUser{}, &ListHostsRequest{})

	if err != nil {
		t.Errorf("expected <<nil>>, got %+v", err)
	}
	if resp.NextPageToken != nextPageToken {
		t.Errorf("expected <<%q>>, got %q", nextPageToken, resp.NextPageToken)
	}
	for i := range resp.Items {
		expected, _ := BuildHostInstance(items[i])
		if !reflect.DeepEqual(resp.Items[i], expected) {
			t.Errorf("unexpected host instance with diff: %s",
				diffPrettyText(prettyJSON(t, *expected), prettyJSON(t, *resp.Items[0])))
		}
	}
}

func TestDeleteHostVerifyUserOwnsTheHost(t *testing.T) {
	var usedListQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usedListQuery = r.URL.Query().Encode()
		replyJSON(w, &compute.InstanceList{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := NewGCEInstanceManager(testConfig, testService, testNameGenerator)

	im.DeleteHost("us-central1-a", &TestUser{}, "foo")

	expected := "alt=json&filter=name%3Dfoo+AND+labels.cf-created_by%3Ajohndoe&prettyPrint=false"
	if usedListQuery != expected {
		t.Errorf("expected <<%q>>, got %q", expected, usedListQuery)
	}
}

func TestDeleteHostHostDoesNotExist(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replyJSON(w, &compute.InstanceList{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := NewGCEInstanceManager(testConfig, testService, testNameGenerator)

	_, err := im.DeleteHost("us-central1-a", &TestUser{}, "foo")

	if appErr, ok := err.(*apperr.AppError); !ok {
		t.Errorf("expected <<%T>>, got %T", appErr, err)
	}
}

func TestDeleteHostSucceeds(t *testing.T) {
	zone := "us-central1-a"
	opName := "operation-1"
	operation := &compute.Operation{
		Name:          opName,
		OperationType: "delete",
		TargetLink:    "https://xyzzy.com/compute/v1/projects/google.com:test-project/zones/us-central1-a/instances/foo",
		Status:        "PENDING",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.URL.Path)
		if r.Method == "DELETE" &&
			r.URL.Path == "/projects/google.com:test-project/zones/us-central1-a/instances/foo" {
			replyJSON(w, operation)
		} else if r.URL.Path == "/projects/google.com:test-project/zones/us-central1-a/instances" {
			replyJSON(w, &compute.InstanceList{Items: []*compute.Instance{{}}})
		} else {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
	}))
	defer ts.Close()
	im := NewGCEInstanceManager(testConfig, buildTestService(t, ts), testNameGenerator)

	op, _ := im.DeleteHost(zone, &TestUser{}, "foo")

	if op.Name != opName {
		t.Errorf("expected <<%q>>, got: %q", opName, op.Name)
	}
}

func TestWaitOperationAndOperationIsNotDone(t *testing.T) {
	zone := "us-central1-a"
	opName := "operation-1"
	operation := &compute.Operation{
		Name:   opName,
		Status: "PENDING",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch path := r.URL.Path; path {
		case "/projects/google.com:test-project/zones/us-central1-a/operations/operation-1/wait":
			replyJSON(w, operation)
		default:
			t.Fatalf("unexpected path: %q", path)
		}
	}))
	defer ts.Close()
	im := NewGCEInstanceManager(testConfig, buildTestService(t, ts), testNameGenerator)

	_, err := im.WaitOperation(zone, &TestUser{}, opName)

	if appErr, _ := err.(*apperr.AppError); true {
		if diff := cmp.Diff(http.StatusServiceUnavailable, appErr.StatusCode); diff != "" {
			t.Errorf("status code (-want +got):\n%s", diff)
		}
	}
}

func TestWaitCreateInstanceOperationSucceeds(t *testing.T) {
	zone := "us-central1-a"
	opName := "operation-1"
	operation := &compute.Operation{
		Name:          opName,
		OperationType: "insert",
		TargetLink:    "https://xyzzy.com/compute/v1/projects/google.com:test-project/zones/us-central1-a/instances/foo",
		Status:        "DONE",
	}
	instance := &compute.Instance{
		Disks:          []*compute.AttachedDisk{{DiskSizeGb: 10}},
		Name:           "foo",
		MachineType:    "mt",
		MinCpuPlatform: "mcp",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch path := r.URL.Path; path {
		case "/projects/google.com:test-project/zones/us-central1-a/operations/operation-1/wait":
			replyJSON(w, operation)
		case "/projects/google.com:test-project/zones/us-central1-a/instances/foo":
			replyJSON(w, instance)
		default:
			t.Fatalf("unexpected path: %q", path)
		}
	}))
	defer ts.Close()
	im := NewGCEInstanceManager(testConfig, buildTestService(t, ts), testNameGenerator)

	res, _ := im.WaitOperation(zone, &TestUser{}, opName)

	want, _ := BuildHostInstance(instance)
	if diff := cmp.Diff(want, res); diff != "" {
		t.Errorf("instance mismatch (-want +got):\n%s", diff)
	}
}

func TestWaitDeleteInstanceOperationSucceeds(t *testing.T) {
	zone := "us-central1-a"
	opName := "operation-1"
	operation := &compute.Operation{
		Name:          opName,
		OperationType: "delete",
		TargetLink:    "https://xyzzy.com/compute/v1/projects/google.com:test-project/zones/us-central1-a/instances/foo",
		Status:        "DONE",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replyJSON(w, operation)
	}))
	defer ts.Close()
	im := NewGCEInstanceManager(testConfig, buildTestService(t, ts), testNameGenerator)

	res, _ := im.WaitOperation(zone, &TestUser{}, opName)

	if diff := cmp.Diff(struct{}{}, res); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestWaitOperationInvalidDoneOperations(t *testing.T) {
	zone := "us-central1-a"
	var operations = map[string]*compute.Operation{
		"oper-1": {Status: "DONE"},
		"oper-2": {
			OperationType: "refresh", // not handled operation type
			TargetLink:    "https://xyzzy.com/compute/v1/projects/google.com:test-project/zones/us-central1-a/instances/foo",
			Status:        "DONE",
		},
		"oper-3": {
			OperationType: "insert",
			// Invalid TargetLink, missing the instance name.
			TargetLink: "https://xyzzy.com/compute/v1/projects/google.com:test-project/zones/us-central1-a/instances/",
			Status:     "DONE",
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for name, op := range operations {
			if strings.HasSuffix(r.URL.Path, name+"/wait") {
				replyJSON(w, op)
				return
			}
		}
		t.Fatalf("unexpected path: %q", r.URL.Path)
	}))
	defer ts.Close()
	im := NewGCEInstanceManager(testConfig, buildTestService(t, ts), testNameGenerator)

	for name := range operations {

		_, err := im.WaitOperation(zone, &TestUser{}, name)

		if err == nil {
			t.Error("expected error")
		}
	}
}

func TestWaitOperationFailedOperation(t *testing.T) {
	zone := "us-central1-a"
	opName := "operation-1"
	errorMessage := "NOT FOUND"
	errorStatusCode := http.StatusNotFound
	operation := &compute.Operation{
		Name:                opName,
		Status:              "DONE",
		Error:               &compute.OperationError{},
		HttpErrorMessage:    errorMessage,
		HttpErrorStatusCode: int64(errorStatusCode),
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replyJSON(w, operation)
	}))
	defer ts.Close()
	im := NewGCEInstanceManager(testConfig, buildTestService(t, ts), testNameGenerator)

	_, err := im.WaitOperation(zone, &TestUser{}, opName)

	appErr, _ := err.(*apperr.AppError)
	if appErr.Msg != errorMessage {
		t.Errorf("expected <<%q>>, got: %q", errorMessage, appErr.Msg)
	}
	if appErr.StatusCode != errorStatusCode {
		t.Errorf("expected <<%d>>, got: %d", errorStatusCode, appErr.StatusCode)
	}
}

func TestBuildHostInstance(t *testing.T) {
	input := &compute.Instance{
		Disks:          []*compute.AttachedDisk{{DiskSizeGb: 10}},
		Name:           "foo",
		MachineType:    "zones/us-central1-a/machineTypes/n1-standard-1",
		MinCpuPlatform: "Intel Haswell",
	}

	got, err := BuildHostInstance(input)

	want := apiv1.HostInstance{
		Name:           "foo",
		BootDiskSizeGB: 10,
		GCP: &apiv1.GCPInstance{
			MachineType:    "n1-standard-1",
			MinCPUPlatform: "Intel Haswell",
		},
	}
	if diff := cmp.Diff(&want, got); diff != "" {
		t.Errorf("instance mismatch (-want +got):\n%s", diff)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestBuildHostInstanceNoDisk(t *testing.T) {
	input := &compute.Instance{
		Disks:          []*compute.AttachedDisk{},
		Name:           "foo",
		MachineType:    "mt",
		MinCpuPlatform: "mcp",
	}

	result, err := BuildHostInstance(input)

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("error type <<\"%T\">> not found in error chain, got %v", appErr, err)
	}
	if result != nil {
		t.Errorf("expected <<nil>>, got %+v", err)
	}
}

func buildTestService(t *testing.T, s *httptest.Server) *compute.Service {
	srv, err := compute.NewService(
		context.TODO(),
		option.WithHTTPClient(s.Client()),
		option.WithEndpoint(s.URL),
	)
	if err != nil {
		t.Fatalf("failed to create compute service withe error: %+v", err)
	}
	return srv
}

func prettyJSON(t *testing.T, obj any) string {
	s, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(s)
}

func diffPrettyText(result string, expected string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(result, expected, false)
	return dmp.DiffPrettyText(diffs)
}

func replyJSON(w http.ResponseWriter, obj any) error {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	return encoder.Encode(obj)
}

type testConstantNameGenerator struct {
	name string
}

func (g *testConstantNameGenerator) NewName() string {
	return g.name
}
