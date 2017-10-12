package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	ca "github.com/hyperledger/fabric-sdk-go/api/apifabca"
	fab "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn"
	deffab "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	"github.com/hyperledger/fabric-sdk-go/def/fabapi/opt"
	"github.com/hyperledger/fabric-sdk-go/pkg/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/events"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/orderer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-txn/admin"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
)

var _LOGGER = log.New(os.Stderr, "SDKClientLogger:", log.Lshortfile)

// SDKClient implementation of setup
type SDKClient struct {
	Client          fab.FabricClient
	Channel         fab.Channel
	EventHub        fab.EventHub
	ConnectEventHub bool
	ConfigFile      string
	OrgID           string
	ChannelID       string
	ChainCodeID     string
	Initialized     bool
	ChannelConfig   string
	AdminUser       ca.User
	OrdererAdmin    ca.User
	SDK             *deffab.FabricSDK
}

// Initialize reads configuration from file and sets up client, channel and event hub
func (sdkClient *SDKClient) Initialize() error {
	_LOGGER.Println("SDKClient.Initialize: start")

	// Create SDK setup for the integration tests
	sdkOptions := deffab.Options{
		ConfigFile: sdkClient.ConfigFile,
		//		OrgID:      setup.OrgID,
		StateStoreOpts: opt.StateStoreOpts{
			Path: "/tmp/enroll_user",
		},
	}

	sdk, err := deffab.NewSDK(sdkOptions)
	if err != nil {
		return fmt.Errorf("Error initializing SDK: %s", err)
	}
	_LOGGER.Println("SDK Created ")
	sdkClient.SDK = sdk
	session, err := sdk.NewPreEnrolledUserSession(sdkClient.OrgID, "Admin")
	if err != nil {
		return fmt.Errorf("Error getting admin user session for org: %s", err)
	}
	sc, err := sdk.NewSystemClient(session)
	if err != nil {
		return fmt.Errorf("NewSystemClient returned error: %v", err)
	}

	sdkClient.Client = sc
	sdkClient.AdminUser = session.Identity()
	_LOGGER.Println("Admin context found ")
	ordererAdmin, err := sdk.NewPreEnrolledUser("ordererorg", "Admin")
	if err != nil {
		return fmt.Errorf("Error getting orderer admin user: %v", err)
	}

	sdkClient.OrdererAdmin = ordererAdmin
	_LOGGER.Println("Orderer admin is retrieved  ")

	sdkClient.Initialized = true

	return nil
}

//InitChannel : Checks if the channel is created with orderer or not else new channel is created
func (sdkClient *SDKClient) InitChannel() (fab.Channel, error) {
	//Check the channel is created in Orderer of not
	channel, err := sdkClient.GetChannel(sdkClient.Client, sdkClient.ChannelID, []string{sdkClient.OrgID})
	if err != nil {
		return nil, fmt.Errorf("Create channel (%s) failed: %v", sdkClient.ChannelID, err)
	}
	_LOGGER.Println("Channel TX file is retrieved  ")
	sdkClient.Channel = channel
	err = channel.Initialize(nil)
	_LOGGER.Printf("Channel is Initialized with Orderer  %v \n", channel.IsInitialized())
	isChannelCreated := true
	if err != nil {
		_LOGGER.Printf("Iniatzation of channel(%s) failed : %v \n", sdkClient.ChannelID, err)
		isChannelCreated = false
	}
	if !isChannelCreated {
		_LOGGER.Println("Going to create channel with admin")
		err = admin.CreateOrUpdateChannel(sdkClient.Client, sdkClient.OrdererAdmin, sdkClient.AdminUser, channel, sdkClient.ChannelConfig)
		if err != nil {
			return nil, fmt.Errorf("Create channel (%s) failed in orderer: %v", sdkClient.ChannelID, err)
		}

		_LOGGER.Println("Waiting for sync")
		time.Sleep(time.Second * 60)

		if err = channel.Initialize(nil); err != nil {
			return nil, fmt.Errorf("Error initializing channel after creation: %v", err)
		}
	}

	return channel, nil
}

//JoinChainnel : This method checks if the peers have joined the channel or not
func (sdkClient *SDKClient) JoinChannel() error {
	foundChannel := false
	for _, peer := range sdkClient.Channel.Peers() {

		_LOGGER.Printf("Checking peer %s\n", peer.URL())
		response, err := sdkClient.Client.QueryChannels(peer)
		if err != nil {
			return fmt.Errorf("Error querying channel for primary peer: %s", err)
		}

		for _, responseChannel := range response.Channels {
			if responseChannel.ChannelId == sdkClient.Channel.Name() {
				foundChannel = true
				_LOGGER.Printf("Peer %s already joined \n", peer.URL())
				_LOGGER.Printf("Peer reponse %v \n", responseChannel)
			}
		}

	}
	if !foundChannel {
		_LOGGER.Printf("Going to join to channel %s \n", sdkClient.ChannelID)
		err := admin.JoinChannel(sdkClient.Client, sdkClient.AdminUser, sdkClient.Channel)
		if err != nil {
			return fmt.Errorf("JoinChannel returned error: %v", err)
		}
	}

	return nil
}

// InitConfig ...
func (sdkClient *SDKClient) InitConfig() (apiconfig.Config, error) {
	configImpl, err := config.InitConfig(sdkClient.ConfigFile)
	if err != nil {
		return nil, err
	}
	return configImpl, nil
}

// GetChannel initializes and returns a channel based on config
func (sdkClient *SDKClient) GetChannel(client fab.FabricClient, channelID string, orgs []string) (fab.Channel, error) {

	channel, err := client.NewChannel(channelID)
	if err != nil {
		return nil, fmt.Errorf("NewChannel return error: %v", err)
	}

	ordererConfig, err := client.Config().RandomOrdererConfig()
	if err != nil {
		return nil, fmt.Errorf("RandomOrdererConfig() return error: %s", err)
	}
	serverHostOverride := ""
	if str, ok := ordererConfig.GRPCOptions["ssl-target-name-override"].(string); ok {
		serverHostOverride = str
	}
	orderer, err := orderer.NewOrderer(ordererConfig.URL, ordererConfig.TLSCACerts.Path, serverHostOverride, client.Config())
	if err != nil {
		return nil, fmt.Errorf("NewOrderer return error: %v", err)
	}

	err = channel.AddOrderer(orderer)
	if err != nil {
		return nil, fmt.Errorf("Error adding orderer: %v", err)
	}

	for _, org := range orgs {

		peerConfig, err := client.Config().PeersConfig(org)
		if err != nil {
			return nil, fmt.Errorf("Error reading peer config: %v", err)
		}
		for _, p := range peerConfig {
			fmt.Println("Parsing peer ", p.EventURL)
			serverHostOverride = ""
			if str, ok := p.GRPCOptions["ssl-target-name-override"].(string); ok {
				serverHostOverride = str
			}
			endorser, err := deffab.NewPeer(p.URL, p.TLSCACerts.Path, serverHostOverride, client.Config())
			if err != nil {
				return nil, fmt.Errorf("NewPeer return error: %v", err)
			}
			err = channel.AddPeer(endorser)
			if err != nil {
				return nil, fmt.Errorf("Error adding peer: %v", err)
			}

		}
	}

	return channel, nil
}

//InstallChainCode install the chain code in the peers associated with the org
func (sdkClient *SDKClient) InstallChainCode(chainCodeID string, chainCodePath string, version string, goPath string) bool {
	err := admin.SendInstallCC(sdkClient.Client, chainCodeID, chainCodePath, version, nil, sdkClient.Channel.Peers(), goPath)
	if err != nil {
		return false
	}
	return true
}
func (sdkClient *SDKClient) InstantiateChainCode(mspID string, chainCodeID string, args [][]byte, path string, version string) bool {
	evtHub, err := sdkClient.getEventHub()
	if err != nil {
		_LOGGER.Fatalf("Error in generating event hub %v\n", err)
		return false
	}
	var trxnTargets []apitxn.ProposalProcessor
	trxnTargets = make([]apitxn.ProposalProcessor, 1)
	for index, peer := range sdkClient.Channel.Peers() {
		trxnTargets[index] = peer
		break
	}
	chaincodePolicy := cauthdsl.SignedByMspMember(mspID)
	err = admin.SendInstantiateCC(sdkClient.Channel, chainCodeID, args, path, version, chaincodePolicy, trxnTargets, evtHub)
	if err != nil {
		_LOGGER.Fatalf("Error instantiate  %v\n", err)
		return false
	}
	return false
}
func (sdkClient *SDKClient) getEventHub() (fab.EventHub, error) {
	eventHub, err := events.NewEventHub(sdkClient.Client)
	if err != nil {
		return nil, fmt.Errorf("NewEventHub failed %v", err)
	}
	foundEventHub := false
	peerConfig, err := sdkClient.Client.Config().PeersConfig(sdkClient.OrgID)
	if err != nil {
		return nil, fmt.Errorf("PeersConfig failed %v ", err)
	}
	for _, p := range peerConfig {
		if p.URL != "" {
			_LOGGER.Printf("EventHub connect to peer (%s)", p.URL)
			serverHostOverride := ""
			if str, ok := p.GRPCOptions["ssl-target-name-override"].(string); ok {
				serverHostOverride = str
			}
			eventHub.SetPeerAddr(p.EventURL, p.TLSCACerts.Path, serverHostOverride)
			foundEventHub = true
			break
		}
	}

	if !foundEventHub {
		return nil, fmt.Errorf("Event hub configuration not found")
	}

	return eventHub, nil
}

// SignedByMspMember creates a SignaturePolicyEnvelope
// requiring 1 signature from any member of the specified MSP
/*func SignedByMspMember(mspIds []string) *cb.SignaturePolicyEnvelope {
	identities := make([]*msp.MSPPrincipal, len(mspIds))
	signatures := make([]*cb.SignaturePolicy, len(mspIds))
	// specify the principal: it's a member of the msp we just found
	for index, mspId := range mspIds {

		principal := &msp.MSPPrincipal{
			PrincipalClassification: msp.MSPPrincipal_ROLE,
			Principal:               MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: mspId})}
		identities[index] = principal
		signatures[index] = cauthdsl.SignedBy(int32(index))
	}
	// create the policy: it requires exactly 1 signature from the first (and only) principal
	p := &cb.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       cauthdsl.NOutOf(int32(len(mspIds)), signatures),
		Identities: identities,
	}

	return p
}
*/

// MarshalOrPanic serializes a protobuf message and panics if this operation fails.
/*func MarshalOrPanic(pb proto.Message) []byte {
	data, err := proto.Marshal(pb)
	if err != nil {
		panic(err)
	}
	return data
}
*/
