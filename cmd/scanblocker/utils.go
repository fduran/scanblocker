package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os/exec"
)

func localIPs() []net.IP {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("Error getting interfaces: %v", err.Error())
	}

	var localIPs []net.IP
	var ip net.IP

	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPAddr:
		case *net.IPNet:
			ip = v.IP
		}

		ip = ip.To4()
		if ip == nil {
			continue
		}

		localIPs = append(localIPs, ip)
	}

	fmt.Printf("localIPs: %v\n", localIPs)
	return localIPs
}

// get ipv4 address from iface (limited to 1)
// credit https://gist.github.com/schwarzeni/f25031a3123f895ff3785970921e962c
func GetInterfaceIpv4Addr(interfaceName string) (addr string, err error) {
	var (
		ief      *net.Interface
		addrs    []net.Addr
		ipv4Addr net.IP
	)

	if ief, err = net.InterfaceByName(interfaceName); err != nil { // get interface
		return
	}

	if addrs, err = ief.Addrs(); err != nil { // get addresses
		return
	}

	for _, addr := range addrs { // get ipv4 address
		if ipv4Addr = addr.(*net.IPNet).IP.To4(); ipv4Addr != nil {
			break // returns first one found
		}
	}

	if ipv4Addr == nil {
		return "", errors.New(fmt.Sprintf("interface %s don't have an ipv4 address\n", interfaceName))
	}

	return ipv4Addr.String(), nil
}

// run OS command
func execute(args []string) {
	_, err := exec.Command(args[0], args[1:]...).Output()

	if err != nil {
		fmt.Printf("Error executing command %v: %v\n", args, err)
	}
}

// try to create a TCP connection, for possible testing
func client(ip string, port int) {
	servAddr := fmt.Sprintf("%s:%v", ip, port)
	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)

	if err != nil {
		println("ResolveTCPAddr failed: ", err.Error())
		return
	}

	localAddress, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:6666")
	conn, err := net.DialTCP("tcp", localAddress, tcpAddr)

	if err != nil {
		// println("Dial failed:", err.Error())
		return
	}

	conn.Close() // doesn't reach
}

// find if an IP is in an array
func containsIP(a []net.IP, ip net.IP) bool {
	for _, v := range a {
		if v.String() == ip.String() { // not the best way
			return true
		}
	}

	return false
}
