package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type Config struct {
	// network device to listen to
	device string // eth0 (Mac: en0, pktap. GCP: ens4)

	// number of ports detected in the last period scanSeconds, minus one
	maxPorts uint16 // 3

	// interval to use when detecting consecutive connections
	scanSeconds int // 60
}

func setOptions() Config {
	config := Config{
		device:      "eth0",  // currently only param as option  
		maxPorts:    3,
		scanSeconds: 60,
	}

	// 1: env vars
	// device
	deviceEnv := os.Getenv("SB_DEVICE")
	if len(deviceEnv) != 0 {
		config.device = deviceEnv
	}

	// 2: CLI args (higher precedence than env vars: run after to override)
	flag.StringVar(&config.device, "i", config.device, "interface to listen to, overrides $SB_DEVICE")
	flag.Parse()

	// TODO: other possible options to add to env vars / arg flags:
	//   Config.device : accept "all"
	//   IP to filter traffic for, if device has multiple IPs
	//   Config.scanSeconds
	//   Config.maxPorts
	//   pcap.OpenLive timeout
	//   pcap filter expression
	//   Prometheus listening port
	//   ability to read from pcap capture file
	//   option to skip call to iptables
	//   verbose logging mode

	return config
}

// listen to network interface & filter
func openListen(config Config) *pcap.Handle {
	// max number of bytes to read: default 262144 = 256k, but the smaller the better (faster)
	// smallest safe number of bytes to read:
	//   eth header: 14 bytes + optional vlan 4 bytes
	//   ip header: 20 + possible options 4 bytes
	//   TCP header 20 bytes but only ports and up to flags: first 14 bytes
	//   total = 56 bytes
	const snapshotLen int32 = 56

	// will not be sniffing all traffic and blocking other nodes in the same LAN segment or WiFi
	const promiscuous bool = false

	// timeout could be made into an option as well
	const timeout time.Duration = pcap.BlockForever

	// Open device
	handle, err := pcap.OpenLive(config.device, snapshotLen, promiscuous, timeout)
	if err != nil {
		log.Fatalf("Error: pcap cannot open device %v: %v", config.device, err)
	}

	// get (first) IP of the iface if exists
	// TODO: return an array of all IPs or use config
	ip, err := GetInterfaceIpv4Addr(config.device)
	if err != nil {
		log.Fatalf("Error: can't get IP address of device %v: %v", config.device, err)
	}

	fmt.Printf("Listening on: %s %s\n", config.device, ip)

	// filter
	// one-way filtering only (incoming connections to host as dst, not host as src)
	// would need to iterate over all possible ips for an iface
	synfilter := "tcp[tcpflags] == tcp-syn and tcp[tcpflags] != tcp-ack" // SYN but not SYN-ACK
	filter := "dst host " + ip + " and " + synfilter                     // + " and src not " + ip
	if err := handle.SetBPFFilter(filter); err != nil {
		log.Fatalf("Error: pcap cannot set filter: %v: %v", filter, err)
	}

	return handle
}

// logic for each packet handled here
func processPacket(config Config, packet gopacket.Packet, ipMap map[string]*Queue,
	disallowedIps *[]net.IP, allowedIps *[]net.IP) {
	tcp, _ := packet.Layer(layers.LayerTypeTCP).(*layers.TCP)
	ip, _ := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)

	tNow := time.Now()
	tUnix := tNow.Unix()
	tFormat := tNow.Format("2006-01-02 15:04:05")

	// print new connection always
	fmt.Printf("%v: New connection: %s:%d -> %s:%d\n", tFormat, ip.SrcIP, tcp.SrcPort, ip.DstIP, int(tcp.DstPort))

	// if ip.SrcIP in allow-list of host local IPs, skip
	if containsIP(*allowedIps, ip.SrcIP) {
		fmt.Printf("Source IP %v in allow-list, skipping\n", ip.SrcIP)
		return
	}

	// add new ports to IP queue and detect scan
	if q, ok := ipMap[ip.SrcIP.String()]; ok {
		// if port already in queue, skip
		for _, con := range q.data {
			if con.port == tcp.DstPort {
				return
			}
		}

		// detect queue full with config.maxPorts different port attempts already in last config.scanSeconds s
		if uint16(len(q.data)) == config.maxPorts && (q.data[0].timestamp >= tUnix-int64(config.scanSeconds)) {
			fmt.Printf("%v: Port scan detected: %s -> %s\n", tFormat, ip.SrcIP, ip.DstIP)

			// add to list and ban with iptables if not in list already
			// note that pcap (therefore this program) will still detect incomming connections from iptables-blocked IPs
			if !containsIP(*disallowedIps, ip.SrcIP) {
				*disallowedIps = append(*disallowedIps, ip.SrcIP)
				// iptables -A (append) seems safer; respecting existing rules (we can for ex allow-list certain IPs)
				commandArgs := strings.Fields("/usr/sbin/iptables -A INPUT -s " + ip.SrcIP.String() + " -j DROP")
				execute(commandArgs)
				fmt.Printf("IP %s banned.\n", ip.SrcIP)
			} else {
				fmt.Printf("IP to ban: %s is already in disallowed list.\n", ip.SrcIP)
			}
		}
		q.Add(conn{tUnix, tcp.DstPort})
	} else {
		// IP not in map, add
		var err error
		ipMap[ip.SrcIP.String()], err = NewQueue(config.maxPorts)
		if err != nil {
			log.Fatalf("Error when creating queue with max port %v: %v", config.maxPorts, err)
		}
		ipMap[ip.SrcIP.String()].Add(conn{tUnix, tcp.DstPort})
	}
}

func main() {
	// get options with env vars or cli flags
	config := setOptions()

	// Run Prometheus in its own (unblocking) goroutine
	prom()

	// open device, listen to traffic & return handle
	handle := openListen(config)
	defer handle.Close()

	// map of IPs to fix-sized "leaky queue" of connections (timestamp, host port)
	// net.IP is a slice, can't be used as map key but we could convert back and forth
	ipMap := make(map[string]*Queue)

	// blacklist of banned IPs
	disallowedIps := make([]net.IP, 0)

	// whitelist of IPs []net.IP
	allowedIps := localIPs()

	// process packets loop
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		// increase Prometheus counter
		synCounter.Inc()

		// process logic for each packet
		processPacket(config, packet, ipMap, &disallowedIps, &allowedIps)
	}
	// unreacheable after loop: don't write code here
}
