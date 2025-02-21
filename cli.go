package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
)

// Selected PeerID
var CurrentNode peer.AddrInfo

// Function for processing commands
func StartCLI() {

	for {
		fmt.Print(CurrentNode.ID, ">")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')

		input = strings.TrimSpace(input)
		parts := strings.Split(input, " ")

		switch parts[0] {

		// use command: use int
		case "use":
			if len(parts) != 2 {
				fmt.Println("Example: use <host id from the providers list>")
				continue
			}

			num, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("Conversion error:", err)
				fmt.Println("Example: use <host id from the providers list>")
				fmt.Println("The id must be int")
				continue
			}

			type Auth struct {
				XMLName  xml.Name `xml:"Auth"`
				Username string   `xml:"Username"`
				Password string   `xml:"Password"`
			}

			auth := Auth{
				Username: "exampleUser",
				Password: "examplePass",
			}

			output, err := xml.MarshalIndent(auth, "", "  ")
			if err != nil {
				log.Fatalf("XML serialization error: %v", err)
				continue
			}

			CurrentNode = Providers[num]
			sendRequestViaMyProtocol(MyHost, ProtocolID, CurrentNode, output)

		case "list":
			if CurrentNode.ID != "" {
				type List struct {
					XMLName xml.Name `xml:"List"`
					Action  string   `xml:"Action"`
				}

				list := List{
					Action: "all",
				}

				output, err := xml.MarshalIndent(list, "", "  ")
				if err != nil {
					log.Fatalf("XML serialization error: %v", err)
					continue
				}

				sendRequestViaMyProtocol(MyHost, ProtocolID, CurrentNode, output)
			} else {
				fmt.Println("Select the host on which you want to print the pods. ")
			}

		case "run":
			if CurrentNode.ID == "" {
				fmt.Println("Select the host on which you want to run pod.")
				continue
			}
			if len(parts) != 4 {
				fmt.Println("Example: run <hash of Pod> <unique ID> <lifetime in hours>")
				continue
			}

			type Run struct {
				XMLName  xml.Name `xml:"Start"`
				Hash     string   `xml:"Hash"`
				UniqueId string   `xml:"UniqueId"`
				Time     string   `xml:"Time"`
			}

			list := Run{
				Hash:     parts[1],
				UniqueId: parts[2],
				Time:     parts[3],
			}

			output, err := xml.MarshalIndent(list, "", "  ")
			if err != nil {
				log.Fatalf("XML serialization error: %v", err)
				continue
			}

			sendRequestViaMyProtocol(MyHost, ProtocolID, CurrentNode, output)

		case "stop":
			if CurrentNode.ID == "" {
				fmt.Println("Select the host on which you want to stop pod.")
				continue
			}

			if len(parts) != 2 {
				fmt.Println("Example: stop <Unique ID>")
				continue
			}

			type Run struct {
				XMLName  xml.Name `xml:"Stop"`
				UniqueId string   `xml:"UniqueId"`
			}

			list := Run{
				UniqueId: parts[1],
			}

			output, err := xml.MarshalIndent(list, "", "  ")
			if err != nil {
				log.Fatalf("XML serialization error: %v", err)
				continue
			}

			sendRequestViaMyProtocol(MyHost, ProtocolID, CurrentNode, output)

		case "running":
			if CurrentNode.ID == "" {
				fmt.Println("Select the host on which you want to print the running pods.")
				continue
			}

			type Running struct {
				XMLName xml.Name `xml:"Running"`
			}

			runngingxml := Running{}

			output, err := xml.MarshalIndent(runngingxml, "", "  ")
			if err != nil {
				log.Fatalf("XML serialization error: %v", err)
				continue
			}

			sendRequestViaMyProtocol(MyHost, ProtocolID, CurrentNode, output)

		case "status":
			if CurrentNode.ID == "" {
				fmt.Println("Select the host on which you want to print the Pod status.")
				continue
			}

			if len(parts) != 2 {
				fmt.Println("Example: status <Unique ID>")
				continue
			}
			type Status struct {
				XMLName  xml.Name `xml:"Status"`
				UniqueId string   `xml:"UniqueId"`
			}

			runngingxml := Status{
				UniqueId: parts[1],
			}

			output, err := xml.MarshalIndent(runngingxml, "", "  ")
			if err != nil {
				log.Fatalf("XML serialization error: %v", err)
				continue
			}

			sendRequestViaMyProtocol(MyHost, ProtocolID, CurrentNode, output)

		case "add":
			if CurrentNode.ID == "" {
				fmt.Println("Select the host on which you want to add the Pod.")
				continue
			}

			if len(parts) != 6 {
				fmt.Println("Example of calling the add command:")
				fmt.Println("add PodName 80 image1,image2 image1 metadata1,metadata2")
				fmt.Println("")
				fmt.Println("PodName —  pod name for convenient identification in the system")
				fmt.Println("80 — numeric value that specifies the internal port of the container")
				fmt.Println("image1,image2 — list of images that will be launched when the pod is started. Values should be specified separated by commas")
				fmt.Println("image1 — image that will look outward. This value must match one of the previous values. All other containers will be available on the Pod's local network.")
				fmt.Println("metadata1,metadata2 — comma separated list of values. This field is optional and is used for convenient identification or sorting of Pods on the host. ")
				continue
			}

			type Pod struct {
				XMLName       xml.Name `xml:"Add"`           // Root element
				PodName       string   `xml:"PodName"`       // Pod name
				Images        []string `xml:"Images>Image"`  // Array of images
				ExternalImage string   `xml:"ExternalImage"` // Image that is accessible externally
				Metadata      []string `xml:"Metadata>Item"` // Array of metadata items
				InternalPort  int      `xml:"InternalPort"`  // Internal port
			}

			num, err := strconv.Atoi(parts[2])
			if err != nil {
				fmt.Println(parts[2])
				fmt.Println("The value must be a number.")
				continue
			}

			runngingxml := Pod{
				PodName:       parts[1],
				InternalPort:  num,
				Images:        strings.Split(parts[3], ","),
				ExternalImage: parts[4],
				Metadata:      strings.Split(parts[5], ","),
			}

			output, err := xml.MarshalIndent(runngingxml, "", "  ")
			if err != nil {
				log.Fatalf("XML serialization error: %v", err)
				continue
			}

			sendRequestViaMyProtocol(MyHost, ProtocolID, CurrentNode, output)

		case "providers":
			for i, p := range Providers {
				fmt.Println(i, p)
			}

		default:
			fmt.Println("providers - Print out all the providers.")
			fmt.Println("use - Select a host for communication.")
			fmt.Println("add - Add Pod to the database on the selected host.")
			fmt.Println("status - Get the status of the Pod by unique identifier on the selected host.")
			fmt.Println("running - View all running pods on the selected host.")
			fmt.Println("run - Run the Pod on the selected host.")
			fmt.Println("stop - Stop the Pod on the selected host.")
			fmt.Println("list - Print all available Pods on the selected host.")

		}

	}
}
