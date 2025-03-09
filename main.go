package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
)

// Chat protocol ID
const chatProtocolID = "/chat/1.0.0"

// Peer Discovery Service Name
const serviceTag = "p2p-chat-service"

// Create a new Libp2p host
func createHost(ctx context.Context) (host.Host, error) {
	return libp2p.New(
		libp2p.DefaultSecurity,
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultPeerstore,
		libp2p.NATPortMap(),
		libp2p.EnableRelay(),
	)
}

// Handle incoming messages
func handleStream(stream network.Stream) {
	reader := bufio.NewReader(stream)
	for {
		msg, err := reader.ReadString(' ')
		if err != nil {
			log.Println("Stream closed:", err)
			return
		}
		fmt.Printf("\n📩 Received: %s", msg)
	}
}

// Setup Peer Discovery using mDNS
func setupMDNSDiscovery(h host.Host) {
	service := mdns.NewMdnsService(h, serviceTag, &discoveryNotifee{h})
	if err := service.Start(); err != nil {
		log.Println("❌ Error starting mDNS discovery:", err)
	} else {
		fmt.Println("✅ mDNS Discovery started...")
	}
}

// Discovery handler
type discoveryNotifee struct {
	host host.Host
}

func (n *discoveryNotifee) HandlePeerFound(peerInfo peer.AddrInfo) {
	fmt.Println("👀 Discovered peer:", peerInfo.ID)
	n.host.Connect(context.Background(), peerInfo)
}

// Send messages to a peer
func sendMessage(h host.Host, peerID peer.ID) {
	stream, err := h.NewStream(context.Background(), peerID, protocol.ID(chatProtocolID))
	if err != nil {
		fmt.Println("❌ Error opening stream:", err)
		return
	}
	writer := bufio.NewWriter(stream)

	for {
		fmt.Print("\n💬 Enter message: ")
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')

		if strings.TrimSpace(text) == "exit" {
			fmt.Println("👋 Exiting chat...")
			return
		}

		writer.WriteString(text)
		writer.Flush()
	}
}

// Setup Kademlia DHT for peer discovery
func setupDHT(ctx context.Context, h host.Host) (*dht.IpfsDHT, error) {
	dhtNode, err := dht.New(ctx, h)
	if err != nil {
		return nil, err
	}
	if err = dhtNode.Bootstrap(ctx); err != nil {
		return nil, err
	}
	fmt.Println("✅ Kademlia DHT initialized...")
	return dhtNode, nil
}

// Main function
func main() {
	ctx := context.Background()

	// Create a new P2P host
	node, err := createHost(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer node.Close()

	fmt.Println("\n🚀 Peer started with ID:", node.ID())
	for _, addr := range node.Addrs() {
		fmt.Println("📡 Listening on:", addr.String()+"/p2p/"+node.ID().String())
	}

	// Enable Peer Discovery
	// var dhtNode *dht.IpfsDHT // Declare dhtNode before assignment
	// var err error             // Declare err before using '='

	setupMDNSDiscovery(node)
	_, err = setupDHT(ctx, node) // ✅ Declare both variables with :=
	if err != nil {
    	   log.Fatal("DHT setup error:", err)
	}

	// Register chat protocol
	node.SetStreamHandler(protocol.ID(chatProtocolID), handleStream)

	// User input loop
	for {
		fmt.Println("\n🔍 Enter peer multiaddress (or 'exit' to quit):")
		reader := bufio.NewReader(os.Stdin)
		peerAddr, _ := reader.ReadString('\n')
		peerAddr = strings.TrimSpace(peerAddr)

		if peerAddr == "exit" {
			fmt.Println("👋 Exiting P2P network...")
			break
		}

		// Parse and connect to the peer
		addr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			fmt.Println("❌ Invalid multiaddr:", err)
			continue
		}
		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			fmt.Println("❌ Could not parse peer address:", err)
			continue
		}

		// Connect to the discovered peer
		err = node.Connect(ctx, *peerInfo)
		if err != nil {
			fmt.Println("❌ Could not connect to peer:", err)
			continue
		}
		fmt.Println("✅ Connected to peer:", peerInfo.ID)

		// Send message to peer
		go sendMessage(node, peerInfo.ID)
	}

	// Allow some time before exiting
	time.Sleep(2 * time.Second)
}
