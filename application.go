package main

import (
	"log"
	"os"
)

func main() {
	var _LOGGER = log.New(os.Stderr, "ApplicationLogger ", log.Lshortfile)
	sdkClient1 := SDKClient{
		ConfigFile:    "config_org1.yaml",
		ChannelID:     "mychannel1",
		OrgID:         "org1",
		ChannelConfig: "/home/suddutt1/fabric_networks_v1/cisco_net/artifacts/channel/mychannel1.tx",
	}
	err := sdkClient1.Initialize()
	if err != nil {
		_LOGGER.Fatalf("Error in initiatzation %v", err)
		return
	}
	_LOGGER.Printf("SDK Initialization Successful for Org 1\n")
	_, err = sdkClient1.InitChannel()
	if err != nil {
		_LOGGER.Fatalf("Error in initiatzation of channel %v\n", err)
		return

	}
	err = sdkClient1.JoinChannel()
	if err != nil {
		_LOGGER.Fatalf("Error in Joining Channel %v\n", err)
		return

	}
	_LOGGER.Printf("Channel Join on org1  successful \n ")

	sdkClient2 := SDKClient{
		ConfigFile:    "config_org2.yaml",
		ChannelID:     "mychannel1",
		OrgID:         "org2",
		ChannelConfig: "/home/suddutt1/fabric_networks_v1/cisco_net/artifacts/channel/mychannel1.tx",
	}
	err = sdkClient2.Initialize()
	if err != nil {
		_LOGGER.Fatalf("Error in initiatzation %v\n", err)
		return
	}
	_LOGGER.Printf("SDK Initialization Successful for Org 2\n")

	sdkClient2.InitChannel()

	/*if err != nil {
		_LOGGER.Fatalf("Error in initiatzation of channel %v\n", err)
		return
	}*/
	err = sdkClient2.JoinChannel()
	if err != nil {
		_LOGGER.Fatalf("Error in Joining Channel %v\n", err)
		return

	}

	_LOGGER.Printf("Channel Join on org2  successful \n ")
	if sdkClient1.InstallChainCode("first", "github.com/suddutt1", "v0", "/home/suddutt1/go2/src/github.com/suddutt1/sdkutilv1/cc") {
		_LOGGER.Println("Chain code installation success at org1 ")
	}
	if sdkClient2.InstallChainCode("first", "github.com/suddutt1", "v0", "/home/suddutt1/go2/src/github.com/suddutt1/sdkutilv1/cc") {
		_LOGGER.Println("Chain code installation success at org2 ")
	}

}
