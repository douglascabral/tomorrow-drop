package main

import (
	"net"
	"net/netip"

	"tomorrow-drop/internal/discovery"
)

// localAddresses returns this device's addresses, as seen by the network stack.
func localAddresses() []netip.Addr {
	return localAddressesFrom(net.InterfaceAddrs, outgoingAddress)
}

// localAddressesFrom uses the interface listing when it works. Some systems,
// Android among them, deny apps the listing, and then the address of the route
// that leads out of the device is used instead.
func localAddressesFrom(list func() ([]net.Addr, error), route func() (netip.Addr, error)) []netip.Addr {
	var out []netip.Addr
	if addrs, err := list(); err == nil {
		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if ip, ok := netip.AddrFromSlice(ipNet.IP); ok {
				out = append(out, ip.Unmap())
			}
		}
	}
	if len(out) > 0 {
		return out
	}
	if ip, err := route(); err == nil && ip.IsValid() {
		return []netip.Addr{ip}
	}
	return nil
}

// outgoingAddress asks the system which local address it would use to reach
// the local network. Connecting a UDP socket sends nothing.
func outgoingAddress() (netip.Addr, error) {
	conn, err := net.DialUDP("udp4", nil, net.UDPAddrFromAddrPort(
		netip.AddrPortFrom(netip.AddrFrom4([4]byte{255, 255, 255, 255}), uint16(discovery.DefaultPort)))) //nolint:gosec // G115: port constant
	if err != nil {
		return netip.Addr{}, err
	}
	defer func() { _ = conn.Close() }()
	udp, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return netip.Addr{}, net.InvalidAddrError("no local address")
	}
	ip, ok := netip.AddrFromSlice(udp.IP)
	if !ok {
		return netip.Addr{}, net.InvalidAddrError("local address is not an IP")
	}
	return ip.Unmap(), nil
}
