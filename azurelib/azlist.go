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

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/davecgh/go-spew/spew"
)

var (
	genericClient    resources.Client
	groupClient      resources.GroupsClient
	addressClient    network.PublicIPAddressesClient
	interfacesClient network.InterfacesClient
	vmClient         compute.VirtualMachinesClient
)

func main() {

	//resourcetypes := [...]string{"Microsoft.Compute/virtualMachines", "Microsoft.DBforPostgreSQL/servers", "Microsoft.Resources/resourceGroups"}
	resourcetypes := [...]string{"Microsoft.Compute/virtualMachines"}
	cl := subscriptions.NewClient()
	// create an authorizer from env vars or Azure Managed Service Idenity
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err == nil {
		cl.Authorizer = authorizer
	}
	l, err := cl.List(context.Background())
	if err != nil {
		panic(err)
	}
	for _, subscription := range l.Values() {
		for _, resourcetype := range resourcetypes {
			filter := fmt.Sprintf("resourceType eq '%s'", resourcetype)
			fmt.Println(*subscription.DisplayName + "\n")
			initializeClients(*subscription.SubscriptionID, authorizer)

			resourceslist, err := genericClient.List(context.Background(), filter, "", nil)
			if err != nil {
				panic(err)
			}
			fmt.Println(len(resourceslist.Values()))
			for _, resource := range resourceslist.Values() {
				spew.Dump(resource)
			}
			// result, err := vmClient.ListAllComplete(context.Background())
			// if err != nil {
			// 	panic(err)
			// }
			groupslist, err := groupClient.List(context.Background(), "", nil)

			if err != nil {
				panic(err)
			}
			for _, v := range groupslist.Values() {
				fmt.Println(*v.Name)

			}

		}
	}
}

func GatherVM(ResourceGroupName string, VMName string) {
	vminfo, err := vmClient.ListComplete(context.Background(), VMName)
	fmt.Println(vminfo)
	if err != nil {
		panic(err)
	}
}

func initializeClients(subscriptionID string, authorizer autorest.Authorizer) {

	genericClient = resources.NewClient(subscriptionID)
	genericClient.Authorizer = authorizer

	groupClient = resources.NewGroupsClient(subscriptionID)
	groupClient.Authorizer = authorizer

	addressClient = network.NewPublicIPAddressesClient(subscriptionID)
	addressClient.Authorizer = authorizer

	interfacesClient = network.NewInterfacesClient(subscriptionID)
	interfacesClient.Authorizer = authorizer

	vmClient = compute.NewVirtualMachinesClient(subscriptionID)
	vmClient.Authorizer = authorizer
}
