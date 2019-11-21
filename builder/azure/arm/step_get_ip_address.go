package arm

import (
	"context"
	"fmt"
    "os"
	"github.com/hashicorp/packer/builder/azure/common/constants"
	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
    "encoding/json"
    "io/ioutil"
)

type EndpointType int

const (
	PublicEndpoint EndpointType = iota
	PrivateEndpoint
	PublicEndpointInPrivateNetwork
)

var (
	EndpointCommunicationText = map[EndpointType]string{
		PublicEndpoint:                 "PublicEndpoint",
		PrivateEndpoint:                "PrivateEndpoint",
		PublicEndpointInPrivateNetwork: "PublicEndpointInPrivateNetwork",
	}
)

type StepGetIPAddress struct {
	client   *AzureClient
	endpoint EndpointType
	get      func(ctx context.Context, resourceGroupName string, ipAddressName string, interfaceName string) (string, error)
	say      func(message string)
	error    func(e error)
}

func NewStepGetIPAddress(client *AzureClient, ui packer.Ui, endpoint EndpointType) *StepGetIPAddress {
	var step = &StepGetIPAddress{
		client:   client,
		endpoint: endpoint,
		say:      func(message string) { ui.Say(message) },
		error:    func(e error) { ui.Error(e.Error()) },
	}

	switch endpoint {
	case PrivateEndpoint:
		step.get = step.getPrivateIP
	case PublicEndpoint:
		step.get = step.getPublicIP
	case PublicEndpointInPrivateNetwork:
		step.get = step.getPublicIPInPrivateNetwork
	}

	return step
}

func (s *StepGetIPAddress) getPrivateIP(ctx context.Context, resourceGroupName string, ipAddressName string, interfaceName string) (string, error) {
	resp, err := s.client.InterfacesClient.Get(ctx, resourceGroupName, interfaceName, "")
	if err != nil {
		s.say(s.client.LastError.Error())
		return "", err
	}

	return *(*resp.IPConfigurations)[0].PrivateIPAddress, nil
}

func (s *StepGetIPAddress) getPublicIP(ctx context.Context, resourceGroupName string, ipAddressName string, interfaceName string) (string, error) {
	resp, err := s.client.PublicIPAddressesClient.Get(ctx, resourceGroupName, ipAddressName, "")
	if err != nil {
		return "", err
	}

	return *resp.IPAddress, nil
}

func (s *StepGetIPAddress) getPublicIPInPrivateNetwork(ctx context.Context, resourceGroupName string, ipAddressName string, interfaceName string) (string, error) {
	s.getPrivateIP(ctx, resourceGroupName, ipAddressName, interfaceName)
	return s.getPublicIP(ctx, resourceGroupName, ipAddressName, interfaceName)
}

func (s *StepGetIPAddress) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Getting the VM's IP address ...")

	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	var ipAddressName = state.Get(constants.ArmPublicIPAddressName).(string)
	var nicName = state.Get(constants.ArmNicName).(string)
	var KeyVaultName = state.Get(constants.ArmKeyVaultName).(string)
	var ComputeName = state.Get(constants.ArmComputeName).(string)

	s.say(fmt.Sprintf(" -> ResourceGroupName   : '%s'", resourceGroupName))
	s.say(fmt.Sprintf(" -> PublicIPAddressName : '%s'", ipAddressName))
	s.say(fmt.Sprintf(" -> NicName             : '%s'", nicName))
	s.say(fmt.Sprintf(" -> Network Connection  : '%s'", EndpointCommunicationText[s.endpoint]))


	address, err := s.get(ctx, resourceGroupName, ipAddressName, nicName)
	if err != nil {
		state.Put(constants.Error, err)
		s.error(err)

		return multistep.ActionHalt
	}

	state.Put(constants.SSHHost, address)
	s.say(fmt.Sprintf(" -> IP Address          : '%s'", address))
    type VmSettings struct {
            IpAddress string `json:"ip_address"`
            NicName  string `json:"nic_name"`
            KeyVaultName  string `json:"keyvault_name"`
            ComputeName string `json:"compute_name"`
        }
    deployment_settings, err := json.Marshal(VmSettings{
        IpAddress: string(address),
        NicName:  string(nicName),
        KeyVaultName:  string(KeyVaultName),
        ComputeName:  string(ComputeName),
    })
    if err != nil {
        panic(err)
    }

    file, _ := json.MarshalIndent(string(deployment_settings), "", " ")
    _ = ioutil.WriteFile(os.Getenv("ARTIFACTSDIRECTORY")+"/packer_deployment.json", file, 0644)



	return multistep.ActionContinue
}

func (*StepGetIPAddress) Cleanup(multistep.StateBag) {
}
