// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package controller_test

import (
	"regexp"

	"github.com/juju/cmd"
	"github.com/juju/cmd/cmdtesting"
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/names.v2"

	"github.com/juju/juju/api/base"
	"github.com/juju/juju/cmd/juju/controller"
	"github.com/juju/juju/core/model"
	"github.com/juju/juju/jujuclient"
	"github.com/juju/juju/jujuclient/jujuclienttesting"
	"github.com/juju/juju/permission"
)

type ShowControllerSuite struct {
	baseControllerSuite
	fakeController *fakeController
	api            func(string) controller.ControllerAccessAPI
}

var _ = gc.Suite(&ShowControllerSuite{})

func (s *ShowControllerSuite) SetUpTest(c *gc.C) {
	s.baseControllerSuite.SetUpTest(c)
	s.fakeController = &fakeController{
		machines: map[string][]base.Machine{
			"ghi": {
				{Id: "0", InstanceId: "id-0", HasVote: false, WantsVote: true, Status: "active"},
				{Id: "1", InstanceId: "id-1", HasVote: false, WantsVote: true, Status: "down"},
				{Id: "2", InstanceId: "id-2", HasVote: true, WantsVote: true, Status: "active"},
				{Id: "3", InstanceId: "id-3", HasVote: false, WantsVote: false, Status: "active"},
			},
		},
	}
	s.api = func(controllerName string) controller.ControllerAccessAPI {
		s.fakeController.controllerName = controllerName
		return s.fakeController
	}
}

func (s *ShowControllerSuite) TestShowOneControllerOneInStore(c *gc.C) {
	s.controllersYaml = `controllers:
  mallards:
    uuid: this-is-another-uuid
    api-endpoints: [this-is-another-of-many-api-endpoints, this-is-one-more-of-many-api-endpoints]
    ca-cert: this-is-another-ca-cert
    cloud: mallards
    agent-version: 999.99.99
`
	s.createTestClientStore(c)

	s.expectedOutput = `
mallards:
  details:
    uuid: this-is-another-uuid
    controller-uuid: this-is-another-uuid
    api-endpoints: [this-is-another-of-many-api-endpoints, this-is-one-more-of-many-api-endpoints]
    ca-cert: this-is-another-ca-cert
    cloud: mallards
    agent-version: 999.99.99
  models:
    controller:
      uuid: abc
      model-uuid: abc
      machine-count: 2
      core-count: 4
    my-model:
      uuid: def
      model-uuid: def
      machine-count: 2
      core-count: 4
  current-model: admin/my-model
  account:
    user: admin
    access: superuser
`[1:]

	s.assertShowController(c, "mallards")
}

func (s *ShowControllerSuite) TestShowControllerWithPasswords(c *gc.C) {
	s.controllersYaml = `controllers:
  mallards:
    uuid: this-is-another-uuid
    api-endpoints: [this-is-another-of-many-api-endpoints, this-is-one-more-of-many-api-endpoints]
    ca-cert: this-is-another-ca-cert
    cloud: mallards
    agent-version: 999.99.99
`
	s.createTestClientStore(c)

	s.expectedOutput = `
mallards:
  details:
    uuid: this-is-another-uuid
    controller-uuid: this-is-another-uuid
    api-endpoints: [this-is-another-of-many-api-endpoints, this-is-one-more-of-many-api-endpoints]
    ca-cert: this-is-another-ca-cert
    cloud: mallards
    agent-version: 999.99.99
  models:
    controller:
      uuid: abc
      model-uuid: abc
      machine-count: 2
      core-count: 4
    my-model:
      uuid: def
      model-uuid: def
      machine-count: 2
      core-count: 4
  current-model: admin/my-model
  account:
    user: admin
    access: superuser
    password: hunter2
`[1:]

	s.assertShowController(c, "mallards", "--show-password")
}

func (s *ShowControllerSuite) TestShowControllerWithBootstrapConfig(c *gc.C) {
	s.controllersYaml = `controllers:
  mallards:
    uuid: this-is-another-uuid
    api-endpoints: [this-is-another-of-many-api-endpoints, this-is-one-more-of-many-api-endpoints]
    ca-cert: this-is-another-ca-cert
    cloud: mallards
    region: mallards1
    agent-version: 999.99.99
`
	store := s.createTestClientStore(c)
	store.BootstrapConfig["mallards"] = jujuclient.BootstrapConfig{
		Config: map[string]interface{}{
			"name":  "controller",
			"type":  "maas",
			"extra": "value",
		},
		Credential:    "my-credential",
		CloudType:     "maas",
		Cloud:         "mallards",
		CloudRegion:   "mallards1",
		CloudEndpoint: "http://mallards.local/MAAS",
	}

	s.expectedOutput = `
mallards:
  details:
    uuid: this-is-another-uuid
    controller-uuid: this-is-another-uuid
    api-endpoints: [this-is-another-of-many-api-endpoints, this-is-one-more-of-many-api-endpoints]
    ca-cert: this-is-another-ca-cert
    cloud: mallards
    region: mallards1
    agent-version: 999.99.99
  models:
    controller:
      uuid: abc
      model-uuid: abc
      machine-count: 2
      core-count: 4
    my-model:
      uuid: def
      model-uuid: def
      machine-count: 2
      core-count: 4
  current-model: admin/my-model
  account:
    user: admin
    access: superuser
`[1:]

	s.assertShowController(c, "mallards")
}

func (s *ShowControllerSuite) TestShowOneControllerManyInStore(c *gc.C) {
	s.createTestClientStore(c)

	s.expectedOutput = `
aws-test:
  details:
    uuid: this-is-the-aws-test-uuid
    controller-uuid: this-is-the-aws-test-uuid
    api-endpoints: [this-is-aws-test-of-many-api-endpoints]
    ca-cert: this-is-aws-test-ca-cert
    cloud: aws
    region: us-east-1
    agent-version: 999.99.99
  controller-machines:
    "0":
      instance-id: id-0
      ha-status: ha-pending
    "1":
      instance-id: id-1
      ha-status: down, lost connection
    "2":
      instance-id: id-2
      ha-status: ha-enabled
  models:
    controller:
      uuid: ghi
      model-uuid: ghi
      machine-count: 2
      core-count: 4
  current-model: admin/controller
  account:
    user: admin
    access: superuser
`[1:]
	s.assertShowController(c, "aws-test")
}

func (s *ShowControllerSuite) TestShowSomeControllerMoreInStore(c *gc.C) {
	s.createTestClientStore(c)
	s.expectedOutput = `
aws-test:
  details:
    uuid: this-is-the-aws-test-uuid
    controller-uuid: this-is-the-aws-test-uuid
    api-endpoints: [this-is-aws-test-of-many-api-endpoints]
    ca-cert: this-is-aws-test-ca-cert
    cloud: aws
    region: us-east-1
    agent-version: 999.99.99
  controller-machines:
    "0":
      instance-id: id-0
      ha-status: ha-pending
    "1":
      instance-id: id-1
      ha-status: down, lost connection
    "2":
      instance-id: id-2
      ha-status: ha-enabled
  models:
    controller:
      uuid: ghi
      model-uuid: ghi
      machine-count: 2
      core-count: 4
  current-model: admin/controller
  account:
    user: admin
    access: superuser
mark-test-prodstack:
  details:
    uuid: this-is-a-uuid
    controller-uuid: this-is-a-uuid
    api-endpoints: [this-is-one-of-many-api-endpoints]
    ca-cert: this-is-a-ca-cert
    cloud: prodstack
    agent-version: 999.99.99
  account:
    user: admin
    access: superuser
`[1:]

	s.assertShowController(c, "aws-test", "mark-test-prodstack")
}

func (s *ShowControllerSuite) TestShowControllerJsonOne(c *gc.C) {
	s.createTestClientStore(c)

	s.expectedOutput = `
{"aws-test":{"details":{"uuid":"this-is-the-aws-test-uuid","api-endpoints":["this-is-aws-test-of-many-api-endpoints"],"ca-cert":"this-is-aws-test-ca-cert","cloud":"aws","region":"us-east-1","agent-version":"999.99.99"},"controller-machines":{"0":{"instance-id":"id-0","ha-status":"ha-pending"},"1":{"instance-id":"id-1","ha-status":"down, lost connection"},"2":{"instance-id":"id-2","ha-status":"ha-enabled"}},"models":{"controller":{"uuid":"ghi","machine-count":2,"core-count":4}},"current-model":"admin/controller","account":{"user":"admin","access":"superuser"}}}
`[1:]

	s.assertShowController(c, "--format", "json", "aws-test")
}

func (s *ShowControllerSuite) TestShowControllerJsonMany(c *gc.C) {
	s.createTestClientStore(c)
	s.expectedOutput = `
{"aws-test":{"details":{"uuid":"this-is-the-aws-test-uuid","api-endpoints":["this-is-aws-test-of-many-api-endpoints"],"ca-cert":"this-is-aws-test-ca-cert","cloud":"aws","region":"us-east-1","agent-version":"999.99.99"},"controller-machines":{"0":{"instance-id":"id-0","ha-status":"ha-pending"},"1":{"instance-id":"id-1","ha-status":"down, lost connection"},"2":{"instance-id":"id-2","ha-status":"ha-enabled"}},"models":{"controller":{"uuid":"ghi","machine-count":2,"core-count":4}},"current-model":"admin/controller","account":{"user":"admin","access":"superuser"}},"mark-test-prodstack":{"details":{"uuid":"this-is-a-uuid","api-endpoints":["this-is-one-of-many-api-endpoints"],"ca-cert":"this-is-a-ca-cert","cloud":"prodstack","agent-version":"999.99.99"},"account":{"user":"admin","access":"superuser"}}}
`[1:]
	s.assertShowController(c, "--format", "json", "aws-test", "mark-test-prodstack")
}

func (s *ShowControllerSuite) TestShowControllerReadFromStoreErr(c *gc.C) {
	s.createTestClientStore(c)

	msg := "fail getting controller"
	errStore := jujuclienttesting.NewStubStore()
	errStore.SetErrors(errors.New(msg))
	s.store = errStore
	s.expectedErr = msg

	s.assertShowControllerFailed(c, "test1")
	errStore.CheckCallNames(c, "ControllerByName")
}

func (s *ShowControllerSuite) TestShowControllerNoArgs(c *gc.C) {
	store := s.createTestClientStore(c)
	store.CurrentControllerName = "aws-test"

	s.expectedOutput = `
{"aws-test":{"details":{"uuid":"this-is-the-aws-test-uuid","api-endpoints":["this-is-aws-test-of-many-api-endpoints"],"ca-cert":"this-is-aws-test-ca-cert","cloud":"aws","region":"us-east-1","agent-version":"999.99.99"},"controller-machines":{"0":{"instance-id":"id-0","ha-status":"ha-pending"},"1":{"instance-id":"id-1","ha-status":"down, lost connection"},"2":{"instance-id":"id-2","ha-status":"ha-enabled"}},"models":{"controller":{"uuid":"ghi","machine-count":2,"core-count":4}},"current-model":"admin/controller","account":{"user":"admin","access":"superuser"}}}
`[1:]
	s.assertShowController(c, "--format", "json")
}

func (s *ShowControllerSuite) TestShowControllerNoArgsNoCurrent(c *gc.C) {
	store := s.createTestClientStore(c)
	store.CurrentControllerName = ""
	s.expectedErr = regexp.QuoteMeta(`there is no active controller`)
	s.assertShowControllerFailed(c)
}

func (s *ShowControllerSuite) TestShowControllerNotFound(c *gc.C) {
	s.createTestClientStore(c)

	s.expectedErr = `controller whoops not found`
	s.assertShowControllerFailed(c, "whoops")
}

func (s *ShowControllerSuite) TestShowControllerUnrecognizedFlag(c *gc.C) {
	s.expectedErr = `option provided but not defined: -m`
	s.assertShowControllerFailed(c, "-m", "my.world")
}

func (s *ShowControllerSuite) TestShowControllerUnrecognizedOptionFlag(c *gc.C) {
	s.expectedErr = `option provided but not defined: --model`
	s.assertShowControllerFailed(c, "--model", "still.my.world")
}

func (s *ShowControllerSuite) TestShowControllerRefreshesStore(c *gc.C) {
	store := s.createTestClientStore(c)
	_, err := s.runShowController(c, "aws-test")
	c.Assert(err, jc.ErrorIsNil)
	c.Check(store.Controllers["aws-test"].ControllerMachineCount, gc.Equals, 3)
	c.Check(store.Controllers["aws-test"].ActiveControllerMachineCount, gc.Equals, 1)
	s.fakeController.machines["ghi"][0].HasVote = true
	_, err = s.runShowController(c, "aws-test")
	c.Assert(err, jc.ErrorIsNil)
	c.Check(store.Controllers["aws-test"].ControllerMachineCount, gc.Equals, 3)
	c.Check(store.Controllers["aws-test"].ActiveControllerMachineCount, gc.Equals, 2)
}

func (s *ShowControllerSuite) TestShowControllerRefreshesStoreModels(c *gc.C) {
	store := s.createTestClientStore(c)
	c.Assert(store.Models["mallards"], gc.DeepEquals, &jujuclient.ControllerModels{
		CurrentModel: "admin/my-model",
		Models: map[string]jujuclient.ModelDetails{
			"model0":   {ModelUUID: "abc", ModelType: model.IAAS},
			"my-model": {ModelUUID: "def", ModelType: model.IAAS},
		},
	})
	_, err := s.runShowController(c, "mallards")
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(store.Models["mallards"], gc.DeepEquals, &jujuclient.ControllerModels{
		CurrentModel: "admin/my-model",
		Models: map[string]jujuclient.ModelDetails{
			"admin/controller": {ModelUUID: "abc", ModelType: model.IAAS},
			"admin/my-model":   {ModelUUID: "def", ModelType: model.IAAS},
		},
	})
}

func (s *ShowControllerSuite) runShowController(c *gc.C, args ...string) (*cmd.Context, error) {
	return cmdtesting.RunCommand(c, controller.NewShowControllerCommandForTest(s.store, s.api), args...)
}

func (s *ShowControllerSuite) assertShowControllerFailed(c *gc.C, args ...string) {
	_, err := s.runShowController(c, args...)
	c.Assert(err, gc.ErrorMatches, s.expectedErr)
}

func (s *ShowControllerSuite) assertShowController(c *gc.C, args ...string) {
	context, err := s.runShowController(c, args...)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(cmdtesting.Stdout(context), gc.Equals, s.expectedOutput)
}

type fakeController struct {
	controllerName string
	machines       map[string][]base.Machine
}

func (*fakeController) GetControllerAccess(user string) (permission.Access, error) {
	return "superuser", nil
}

func (*fakeController) ModelConfig() (map[string]interface{}, error) {
	return map[string]interface{}{"agent-version": "999.99.99"}, nil
}

func (c *fakeController) ModelStatus(models ...names.ModelTag) (result []base.ModelStatus, _ error) {
	for _, mtag := range models {
		result = append(result, base.ModelStatus{
			UUID:              mtag.Id(),
			TotalMachineCount: 2,
			CoreCount:         4,
			Machines:          c.machines[mtag.Id()],
		})
	}
	return result, nil
}

func (c *fakeController) AllModels() (result []base.UserModel, _ error) {
	models := map[string][]base.UserModel{
		"aws-test": {
			{Name: "controller", UUID: "ghi", Owner: "admin", Type: model.IAAS},
		},
		"mallards": {
			{Name: "controller", UUID: "abc", Owner: "admin", Type: model.IAAS},
			{Name: "my-model", UUID: "def", Owner: "admin", Type: model.IAAS},
		},
	}
	all, exists := models[c.controllerName]
	if !exists {
		return result, nil
	}
	return all, nil
}

func (*fakeController) Close() error {
	return nil
}
