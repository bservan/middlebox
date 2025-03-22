package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/nats-io/nats.go"
)

// Custom data structure that holds NATS connection handle and average packet processing delay
type PacketProcessorContext struct {
	natsConn       *nats.Conn
	averageDelayMs int
}

// Creates PacketProcessorContext
func NewPacketProcessorContext(nc *nats.Conn, averageDelayMs int) *PacketProcessorContext {
	return &PacketProcessorContext{
		natsConn:       nc,
		averageDelayMs: averageDelayMs,
	}
}

// Will sleep int [0, 2 * avg delay] interval (in milliseconds).
// Sleep amount generation is cryptographically secure.
func (ctx *PacketProcessorContext) addRandomDelay() {
	rangeSize := 2*ctx.averageDelayMs + 1
	bignum, _ := rand.Int(rand.Reader, big.NewInt(int64(rangeSize)))
	delayMs := int(bignum.Int64())
	time.Sleep(time.Millisecond * time.Duration(delayMs))
}

// Function to process the ethernet packet
func (ctx *PacketProcessorContext) processEthernetPacket(iface string, data []byte) {
	// Add your ethernet packet processing logic here
	fmt.Printf("Processing ethernet packet: %s\n", iface)

	// Use gopacket to dissect the packet
	packet := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.Default)
	if packet.ErrorLayer() != nil {
		fmt.Println("Error decoding some part of the packet:", packet.ErrorLayer().Error())
		return
	}
	// Check for Ethernet layer
	if ethernetLayer := packet.Layer(layers.LayerTypeEthernet); ethernetLayer != nil {
		fmt.Println("Ethernet layer detected.")
		fmt.Println(gopacket.LayerDump(ethernetLayer))
	}

	// Check for IP layer
	if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		fmt.Println("IPv4 layer detected.")
		fmt.Println(gopacket.LayerDump(ipLayer))
	} else if ipLayer := packet.Layer(layers.LayerTypeIPv6); ipLayer != nil {
		fmt.Println("IPv6 layer detected.")
		fmt.Println(gopacket.LayerDump(ipLayer))
	}

	// Check for TCP layer
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		fmt.Println("TCP layer detected.")
		fmt.Println(gopacket.LayerDump(tcpLayer))
	}

	// Check for UDP layer
	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		fmt.Println("UDP layer detected.")
		fmt.Println(gopacket.LayerDump(udpLayer))
	}
	// Publish the processed packet to the appropriate subject
	var subject string
	if iface == "inpktsec" {
		subject = "outpktinsec"
	} else {
		subject = "outpktsec"
	}
	// Add random delay (in ms)
	ctx.addRandomDelay()
	err := ctx.natsConn.Publish(subject, data)
	if err != nil {
		fmt.Println("Error publishing message:", err)
	}
}

func (ctx *PacketProcessorContext) subscribeToSubjects() {
	ctx.natsConn.Subscribe("inpktsec", func(m *nats.Msg) {
		//fmt.Printf("Received a message: %s\n", string(m.Data))
		// Process the incoming ethernet packet here
		ctx.processEthernetPacket(m.Subject, m.Data)
	})

	// Simple Subscriber
	ctx.natsConn.Subscribe("inpktinsec", func(m *nats.Msg) {
		//fmt.Printf("Received a message: %s\n", string(m.Data))
		// Process the incoming ethernet packet here
		ctx.processEthernetPacket(m.Subject, m.Data)
	})
}

func main() {
	fmt.Println("Hello, World!")
	url := os.Getenv("NATS_SURVEYOR_SERVERS")
	if url == "" {
		url = nats.DefaultURL
	}
	fmt.Println("NATS_SURVEYOR_SERVERS: ", url)

	averageDelayMs := 0
	averageDelayStr := os.Getenv("MIDDLEBOX_PROCESSOR_AVG_DELAY_MS")
	if averageDelayStr != "" {
		delayMs, err := strconv.Atoi(averageDelayStr)
		if err == nil && delayMs >= 0 {
			averageDelayMs = delayMs
		}
	}
	fmt.Printf("MIDDLEBOX_PROCESSOR_AVG_DELAY_MS: %d ms.\n", averageDelayMs)

	// Connect to a server
	nc, _ := nats.Connect(url)
	defer nc.Drain()

	// Create packet processor context
	pktProcessorCtx := NewPacketProcessorContext(nc, averageDelayMs)

	// Subscribe to subjects
	pktProcessorCtx.subscribeToSubjects()

	// Keep the connection alive
	select {}

	// Drain connection (Preferred for responders)
	// Close() not needed if this is called.

	// Close connection
	nc.Close()
}
