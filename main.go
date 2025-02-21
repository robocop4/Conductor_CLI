package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/pnet"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"

	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
)

var Providers []peer.AddrInfo
var ProtocolID protocol.ID
var MyHost host.Host

// Connection manager to limit connections
var connMgr, _ = connmgr.NewConnManager(1, 2, connmgr.WithGracePeriod(time.Minute))

// Extra options for libp2p
var Libp2pOptionsExtra = []libp2p.Option{
	libp2p.NATPortMap(),
	libp2p.ConnectionManager(connMgr),
	//libp2p.EnableAutoRelay(),
	libp2p.EnableNATService(),
}

func handleConnection(net network.Network, conn network.Conn) {

	// Here you can reject the connection based on the blacklist

}

func main() {
	var dhtUid string

	flag.StringVar(&dhtUid, "cid", "", "SID of Conductor")

	flag.Parse()

	if dhtUid == "" {
		fmt.Println("Use the flag --cid")
		return
	}

	ProtocolID = "/conductor/0.0.1"
	ctx := context.Background()
	privKey, _ := LoadKeyFromFile()
	listen, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")

	// Initialize our host
	h, mydht, _, err := SetupLibp2p(
		ctx,
		privKey,
		nil,
		[]multiaddr.Multiaddr{listen},
		nil,
		Libp2pOptionsExtra...,
	)
	if err != nil {
		log.Panic(err.Error())

	}
	defer h.Close()

	MyHost = h
	fmt.Println("My id: ", h.ID().String())
	fmt.Println("My address: ", h.Addrs())

	h.Network().Notify(&network.NotifyBundle{
		ConnectedF: handleConnection,
	})

	// Connect to a known host
	bootstrapHost, _ := multiaddr.NewMultiaddr("/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ")
	peerinfo, _ := peer.AddrInfoFromP2pAddr(bootstrapHost)
	err = h.Connect(ctx, *peerinfo)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	time.Sleep(5 * time.Second)

	routingTable := mydht.RoutingTable()
	log.Printf("DHT routing table size: %d", routingTable.Size())

	provideCid := cid.NewCidV1(cid.Raw, []byte(dhtUid))

	// Find providers for the given CID
	Providers, err = mydht.FindProviders(ctx, provideCid)
	if err != nil {
		log.Fatalf("Failed to find providers: %v", err)
	}

	StartCLI()

}

// Configuring libp2p
func SetupLibp2p(ctx context.Context,
	hostKey crypto.PrivKey,
	secret pnet.PSK,
	listenAddrs []multiaddr.Multiaddr,
	ds datastore.Batching,
	opts ...libp2p.Option) (host.Host, *dht.IpfsDHT, peer.ID, error) {
	var ddht *dht.IpfsDHT

	var err error
	var transports = libp2p.DefaultTransports
	//var transports = libp2p.NoTransports
	if secret != nil {
		transports = libp2p.ChainOptions(
			libp2p.NoTransports,
			libp2p.Transport(tcp.NewTCPTransport),
			//libp2p.Transport(websocket.New),
		)
	}

	finalOpts := []libp2p.Option{
		libp2p.Identity(hostKey),
		libp2p.ListenAddrs(listenAddrs...),
		libp2p.PrivateNetwork(secret),
		transports,
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			ddht, err = newDHT2(ctx, h, ds)
			return ddht, err
		}),
	}
	finalOpts = append(finalOpts, opts...)

	h, err := libp2p.New(
		finalOpts...,
	)
	if err != nil {
		return nil, nil, "", err
	}

	pid, _ := peer.IDFromPublicKey(hostKey.GetPublic())
	// Connect to default peers

	return h, ddht, pid, nil
}

// Create a new DHT instance
func newDHT2(ctx context.Context, h host.Host, ds datastore.Batching) (*dht.IpfsDHT, error) {
	var options []dht.Option

	// If no bootstrap peers, this peer acts as a bootstrapping node
	// Other peers can use this peer's IPFS address for peer discovery via DHT
	options = append(options, dht.Mode(dht.ModeAuto))

	kdht, err := dht.New(ctx, h, options...)
	if err != nil {
		return nil, err
	}

	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	return kdht, nil
}

// Load keys from file or generate new ones
func LoadKeyFromFile() (crypto.PrivKey, crypto.PubKey) {
	privKey, err := os.ReadFile("key.priv")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			privKey, pubKey := saveKeyToFile()
			return privKey, pubKey
		} else {
			panic(err)
		}
	}

	pubKey, err := os.ReadFile("key.pub")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			privKey, pubKey := saveKeyToFile()
			return privKey, pubKey
		} else {
			panic(err)
		}
	}
	privKeyByte, err := crypto.UnmarshalPrivateKey(privKey)
	if err != nil {
		panic(err)
	}
	pubKeyByte, err := crypto.UnmarshalPublicKey(pubKey)
	if err != nil {
		panic(err)
	}
	return privKeyByte, pubKeyByte
}

// Save generated keys to files
func saveKeyToFile() (crypto.PrivKey, crypto.PubKey) {
	fmt.Println("[+] Generate keys")
	privKey, pubKey, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		panic(err)
	}
	privKeyByte, _ := crypto.MarshalPrivateKey(privKey)
	pubKeyByte, _ := crypto.MarshalPublicKey(pubKey)
	err = os.WriteFile("key.priv", privKeyByte, 0644)
	if err != nil {
		// Handle error (optional)
	}
	err = os.WriteFile("key.pub", pubKeyByte, 0644)
	if err != nil {
		panic(err)
	}
	return privKey, pubKey
}

func sendRequestViaMyProtocol(h host.Host, ProtocolID protocol.ID, peerAddr peer.AddrInfo, body []byte) {
	// Connecting to a remote node
	if err := h.Connect(context.Background(), peer.AddrInfo{ID: peerAddr.ID, Addrs: peerAddr.Addrs}); err != nil {
		log.Fatal("Connection failed:", err)
		return
	}

	// Sending HTTP request via libp2p
	stream, err := h.NewStream(context.Background(), peerAddr.ID, ProtocolID)
	if err != nil {
		log.Fatal("Error creating stream:", err)
		return
	}

	// Example of sending HTTP request via libp2p stream
	_, err = stream.Write(body)
	if err != nil {
		log.Fatal("Error writing to stream:", err)
		return
	}

	// Now, let's wait for a response from the remote peer
	// Reading response from the stream
	response := make([]byte, 1024) // Buffer to store the response
	n, err := stream.Read(response)
	if err != nil && err != io.EOF {
		log.Fatal("Error reading from stream:", err)
		return
	}

	// Handle the response
	fmt.Printf("Received response: %s\n", string(response[:n]))

	// Close the stream after reading the response
	err = stream.Close()
	if err != nil {
		log.Fatal("Error closing stream:", err)
		return
	}

}
