/*
Copyright 2019 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/pkg/errors"
)

var (
	genericClient    resources.Client
	groupClient      resources.GroupsClient
	addressClient    network.PublicIPAddressesClient
	interfacesClient network.InterfacesClient
	vmClient         compute.VirtualMachinesClient
)

//AzureSession - struct that holds the Azure Session. Goal is to pass it to multiple methods that require authorization
type AzureSession struct {
	SubscriptionID string
	Authorizer     autorest.Authorizer
}

//NicDescriber - struct that holds data such as public, private IP and VM info from the NIC
type NicDescriber struct {
	PrivateIP *string
	NicName   string
	//PublicIP  *network.PublicIPAddress
	//Vmachine string
}

//NicDescriberList - struct to hold a slice of NicDescribers
type NicDescriberList struct {
	NicDL []NicDescriber
}

// AddItem - Helper function to add an item to the slice
func (Ndl *NicDescriberList) AddItem(Nd NicDescriber) []NicDescriber {
	Ndl.NicDL = append(Ndl.NicDL, Nd)
	return Ndl.NicDL
}

func newSession() (*AzureSession, error) {

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	subscriptionID := "282160c0-3c83-43f1-bff1-9356b1678ffb"

	if err != nil {
		fmt.Println("error from session init", err)
	}

	sess := AzureSession{
		SubscriptionID: subscriptionID,
		Authorizer:     authorizer,
	}
	return &sess, nil
}

func getGroups(sess *AzureSession) ([]string, error) {
	listResgroups := make([]string, 0)
	var err error

	grClient := resources.NewGroupsClient(sess.SubscriptionID)
	grClient.Authorizer = sess.Authorizer

	for list, err := grClient.ListComplete(context.Background(), "", nil); list.NotDone(); err = list.Next() {
		if err != nil {
			return nil, errors.Wrap(err, "error traverising RG list")
		}
		rgName := *list.Value().Name
		listResgroups = append(listResgroups, rgName)
	}
	//fmt.Println(listResgroups)
	return listResgroups, err
}

func getNics(sess *AzureSession, rgName string) ([]NicDescriber, error) {

	//listNics := make([]string, 0)
	var NDList NicDescriberList
	var err error

	nwClient := network.NewInterfacesClient(sess.SubscriptionID)
	nwClient.Authorizer = sess.Authorizer

	for nwList, nwErr := nwClient.ListComplete(context.Background(), rgName); nwList.NotDone(); nwErr = nwList.Next() {
		if nwErr != nil {
			return nil, errors.Wrap(err, "error parsing the Nics")
		}

		//fmt.Printf("%T", nwList)

		//listNics = append(listNics, eachNic)
		//fmt.Println("eachNic", eachNic)
		eachNic := *nwList.Value().Name
		ipConfigs := *nwList.Value().IPConfigurations
		for _, eachipCfg := range ipConfigs {
			item := NicDescriber{
				PrivateIP: eachipCfg.PrivateIPAddress,
				NicName:   eachNic,
			}
			NDList.AddItem(item)
		}
		//fmt.Println("each IP config -- ", eachipConfig)

	}

	return NDList.NicDL, err

}

func main() {
	sess, err := newSession()

	if err != nil {
		fmt.Printf("cannot obtain the session -- %v\n", err)
		os.Exit(1)
	}

	groups, groupErr := getGroups(sess)
	if groupErr != nil {
		fmt.Println("unable to retrieve Resource Groups -- ", groupErr)
	}

	fmt.Println("list of Resource Groups -- ", groups)

	for _, eachGrp := range groups {
		nicdata, nicErr := getNics(sess, eachGrp)
		if nicErr != nil {
			fmt.Println("unable to retrieve NIC list ", nicErr)
		}

		for _, vmInNic := range nicdata {
			fmt.Printf(" Private IP Address of NIC: %v is  - %v \n ", vmInNic.NicName, *vmInNic.PrivateIP)
		}

	}
}
