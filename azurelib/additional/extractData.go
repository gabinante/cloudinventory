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
	"encoding/json"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/azure-sdk-for-go/services/postgresql/mgmt/2017-12-01-preview/postgresql"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-04-01/resources"
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
	NicName   string
	PrivateIP string
	PublicIP  string
	Vmachine  string
}

//NicDescriberList - struct to hold a slice of NicDescribers
type NicDescriberList struct {
	NicDL []NicDescriber
}

// NWIntPubIP - struct that stores the PublicIP's ID
type NWIntPubIP struct {
	ID string
}

// NWIntProp - struct that stores the network interface "properties" field
type NWIntProp struct {
	PrivateIPAddress          string
	PrivateIPAllocationMethod string
	PrivateIPAddressVersion   string
	PublicIPAddress           NWIntPubIP
}

// NWIntConf - struct that stores the Network Interface response
type NWIntConf struct {
	Etag       string
	ID         string
	Name       string
	Properties NWIntProp
}

type nwIntRes struct {
	Name       string
	location   string
	Properties nwIntResProp
}

type nwIntResProp struct {
	VirtualMachine nwIntResVM
}

type nwIntResVM struct {
	ID string
}

// PGServer - struct to store data for each postgres server
type PGServer struct {
	ID         string
	Properties PGServerProperties
	Location   string
	Name       string
	Type       string
}

// PGServerProperties - struct to display "properties" of each PGServer
type PGServerProperties struct {
	AdministratorLogin       string
	StorageProfile           PGStorageProfile
	FullyQualifiedDomainName string
	Version                  string
}

// PGStorageProfile - captures storage profile of the PG server
type PGStorageProfile struct {
	StorageMB int64
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
		var item NicDescriber

		//fmt.Println(eachNic, "-----", len(*nwList.Response().Value))
		//fmt.Printf("%T\n", *nwList.Response().Value)
		//fmt.Println(reflect.ValueOf(*nwList.Response().Value))

		//fmt.Printf("length of IP configurations of NIC %s is %v \n", eachNic, len(*nwList.Value().IPConfigurations))
		//fmt.Printf("IP configuration %v\n", *nwList.Value().IPConfigurations)

		var vmBS []byte
		var vmErr error

		vmBS, vmErr = nwList.Value().MarshalJSON()
		if vmErr != nil {
			fmt.Println("cannot unmarshal reponse ")
		}
		var thisNwIntRes nwIntRes
		//fmt.Println("Network INt response for VM is ", string(vmBS))
		json.Unmarshal(vmBS, &thisNwIntRes)

		for _, val1 := range *nwList.Value().IPConfigurations {
			//fmt.Println(val1)
			//fmt.Printf("%T,\n", val1)
			var bs []byte
			var errNw error
			bs, errNw = val1.MarshalJSON()
			if errNw != nil {
				fmt.Println("cannot Marhsal JSON")
			}
			//fmt.Println(string(bs))

			//var jsonRes map[string]interface{}
			//var thisPrivateIP string
			//var thisPublicIP string
			var thisNWInt NWIntConf
			json.Unmarshal(bs, &thisNWInt)

			item = NicDescriber{
				PrivateIP: thisNWInt.Properties.PrivateIPAddress,
				PublicIP:  thisNWInt.Properties.PublicIPAddress.ID,
				NicName:   eachNic,
				Vmachine:  thisNwIntRes.Properties.VirtualMachine.ID,
			}
			NDList.AddItem(item)

		}

	}

	return NDList.NicDL, err

}

func describeResource(sess *AzureSession, rgID string) (resources.GenericResource, error) {
	resourcesClient := resources.NewClient(sess.SubscriptionID)
	resourcesClient.Authorizer = sess.Authorizer

	resData, resErr := resourcesClient.GetByID(context.Background(), rgID)

	return resData, resErr
}

func extractIPfromGenRes(genRes resources.GenericResource) (string, error) {
	var bs []byte
	var err error
	type GenProps struct {
		IPAddress string
	}
	type GenResType struct {
		ID         string
		Name       string
		Properties GenProps
	}

	var GenRes1 GenResType

	bs, err = genRes.MarshalJSON()
	json.Unmarshal(bs, &GenRes1)
	//fmt.Println(string(bs))

	return GenRes1.Properties.IPAddress, err

}

func extractVMfromGenRes(genRes resources.GenericResource) (string, error) {
	var bs []byte
	var err error

	type GenResType struct {
		ID   string
		Name string
	}

	var GenRes1 GenResType

	bs, err = genRes.MarshalJSON()
	json.Unmarshal(bs, &GenRes1)
	//fmt.Println(string(bs))

	return GenRes1.Name, err

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

	//fmt.Println("list of Resource Groups -- ", groups)

	for _, eachGrp := range groups {
		nicdata, nicErr := getNics(sess, eachGrp)
		if nicErr != nil {
			fmt.Println("unable to retrieve NIC list ", nicErr)
		}

		for _, vmInNic := range nicdata {

			if len(vmInNic.PublicIP) > 0 {
				res, err := describeResource(sess, vmInNic.PublicIP)
				if err != nil {
					fmt.Println("cannot describe resource - ", vmInNic.PublicIP)
					fmt.Println(err)
				}
				res1, err1 := extractIPfromGenRes(res)
				if err != nil {
					fmt.Println("cannot extract IP address ", err1)
				}
				//fmt.Println(res1)
				vmInNic.PublicIP = res1
				if len(res1) == 0 {
					vmInNic.PublicIP = "Public IP not created"
				}

			} else if len(vmInNic.PublicIP) == 0 {
				vmInNic.PublicIP = "Public IP not created"

			}
			if len(vmInNic.Vmachine) > 0 {
				res, err := describeResource(sess, vmInNic.Vmachine)
				if err != nil {
					fmt.Println("cannot describe resource - ", vmInNic.Vmachine)
					fmt.Println(err)
				}
				res1, err1 := extractVMfromGenRes(res)
				if err1 != nil {
					fmt.Println("cannot extract VM details ", err1)
				}
				vmInNic.Vmachine = res1
				if len(res1) == 0 {
					vmInNic.Vmachine = "VM not assigned"
				}

			} else if len(vmInNic.Vmachine) == 0 {
				vmInNic.Vmachine = "VM not assigned"
			}
			//fmt.Printf(" NIC: %v -- VM name:  %v \n", vmInNic.NicName, vmInNic.Vmachine)
			//fmt.Printf(" NIC: %v -- Private IP: %v \n ", vmInNic.NicName, vmInNic.PrivateIP)
			//fmt.Printf(" NIC: %v -- Public IP: %v \n", vmInNic.NicName, vmInNic.PublicIP)
			fmt.Printf("%v \n", vmInNic)

		}

	}

	postgresqlClient := postgresql.NewServersClient(sess.SubscriptionID)
	postgresqlClient.Authorizer = sess.Authorizer

	var pgServer PGServer

	pgList, pgErr := postgresqlClient.List(context.Background())
	if pgErr != nil {
		fmt.Println(pgErr)
	}

	fmt.Println(" ========== POSTGRES DATA =========== ")
	for _, eachpgServer := range *pgList.Value {
		var bs []byte
		var err error

		bs, err = eachpgServer.MarshalJSON()
		if err != nil {
			fmt.Println("cant make sense of pg server list ", err)
		}

		json.Unmarshal(bs, &pgServer)

		fmt.Printf("Name: %v -- ID: %v -- Version: %v -- size: %v \n\n", pgServer.Name, pgServer.ID, pgServer.Properties.Version, pgServer.Properties.StorageProfile.StorageMB)

	}

}
