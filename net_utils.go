package netaddr

import (
	"bytes"
	"fmt"
	"math/big"
	"net"
	"strings"
)

type IPNet struct {
	*net.IPNet
}

type NetworkOperations interface {
	Size() *big.Int
	NetworkAddr() net.IP
	BroadcastAddr() net.IP
	ContainsNet(m *IPNet) bool
	Difference(m *IPNet) []*IPNet
	DivideInHalf() (a, b *IPNet)
	CanCombineWith(m *IPNet) (ok bool, newNet *IPNet)
	Expand(limit int) []net.IP
}

// NetSize returns the size of the given IPNet in terms of the number of
// addresses. It always includes the network and broadcast addresses.
func NetSize(n *net.IPNet) *big.Int {
	ipNet := IPNet{n}
	return ipNet.Size()
}

// IPNetSize returns the size of the given IPNet in terms of the number of
// addresses. It always includes the network and broadcast addresses.
func (n *IPNet) Size() *big.Int {
	ones, bits := n.Mask.Size()
	return big.NewInt(0).Lsh(big.NewInt(1), uint(bits-ones))
}

// ParseIP is like net.ParseIP except that it parses IPv4 addresses as 4 byte
// addresses instead of 16 bytes mapped IPv6 addresses. This has been one of my
// biggest gripes against the net package.
func ParseIP(address string) net.IP {
	if strings.Contains(address, ":") {
		return net.ParseIP(address)
	}
	return net.ParseIP(address).To4()
}

// ParseNet parses an IP network from a CIDR. Unlike net.ParseCIDR, it does not
// allow a CIDR where the host part is non-zero. For example, the following
// CIDRs will result in an error: 203.0.113.1/24, 2001:db8::1/64, 10.0.20.0/20
func ParseNet(cidr string) (parsed *net.IPNet, err error) {
	ip, parsed, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	if !ip.Equal(parsed.IP) {
		err = fmt.Errorf("host part is not zero")
		return nil, err
	}
	return
}

// ParseIPNet parses an IP network from a CIDR. Unlike net.ParseCIDR, it does not
// allow a CIDR where the host part is non-zero. For example, the following
// CIDRs will result in an error: 203.0.113.1/24, 2001:db8::1/64, 10.0.20.0/20
func ParseIPNet(cidr string) (parsed *IPNet, err error) {
	parsedNet, err := ParseNet(cidr)
	if err != nil {
		return nil, err
	}
	return &IPNet{parsedNet}, err
}

// NewIP returns a new IP with the given size. The size must be 4 for IPv4 and
// 16 for IPv6.
func NewIP(size int) net.IP {
	if size == 4 {
		return net.ParseIP("0.0.0.0").To4()
	}
	if size == 16 {
		return net.ParseIP("::")
	}
	panic("Bad value for size")
}

// NetworkAddr returns the first address in the given network, or the network address.
func NetworkAddr(n *net.IPNet) net.IP {
	return n.IP
}

// NetworkAddr returns the first address in the given network, or the network address.
func (n *IPNet) NetworkAddr() net.IP {
	return n.IP
}

// BroadcastAddr returns the last address in the given network, or the broadcast address.
func BroadcastAddr(n *net.IPNet) net.IP {
	ipNet := IPNet{n}
	return ipNet.BroadcastAddr()
}

// BroadcastAddr returns the last address in the given network, or the broadcast address.
func (n *IPNet) BroadcastAddr() net.IP {
	// The golang net package doesn't make it easy to calculate the broadcast address. :(
	broadcast := NewIP(len(n.IP))
	for i := 0; i < len(n.IP); i++ {
		broadcast[i] = n.IP[i] | ^n.Mask[i]
	}
	return broadcast
}

// ContainsNet returns true if inner is a subset of n. To be clear, it
// returns true if n == inner also.
func (n *IPNet) ContainsNet(inner *IPNet) bool {
	// If the two networks are different IP versions, return false
	if len(n.IP) != len(inner.IP) {
		return false
	}
	if !n.IPNet.Contains(inner.IP) {
		return false
	}
	if !n.IP.Equal(inner.IP) {
		return true
	}
	return bytes.Compare(n.Mask, inner.Mask) <= 0
}

// Difference returns the set difference n - m. It returns the list of CIDRs
// in order from largest to smallest. They are *not* sorted by network IP.
func (n *IPNet) Difference(m *IPNet) (result []*IPNet) {
	// If the two networks are different IP versions, return n
	if len(n.IP) != len(m.IP) {
		return []*IPNet{n}
	}

	// If m contains n then the difference is empty
	if m.ContainsNet(n) {
		return
	}
	// If n doesn't contain m then the difference is equal to n
	if !n.ContainsNet(m) {
		return []*IPNet{n}
	}

	// If two nets overlap then one must contain the other. At this point, we
	// know n contains m and m is smaller than n. Cut n in half and recurse on
	// the one that overlaps
	first, second := n.DivideInHalf()
	if bytes.Compare(m.IP, second.IP) < 0 {
		return append([]*IPNet{second}, first.Difference(m)...)
	}
	return append([]*IPNet{first}, second.Difference(m)...)
}

// DivideInHalf returns the given net as two equally sized halves
func (n *IPNet) DivideInHalf() (a, b *IPNet) {
	// Get the size of the original netmask
	ones, bits := n.Mask.Size()

	// Netmask has one more 1. Net is half the size of original.
	mask := net.CIDRMask(ones+1, bits)

	// Create a new IP to fill in for the second half
	ip := net.ParseIP("::")
	if bits == 32 {
		ip = net.ParseIP("0.0.0.0").To4()
	}
	// Fill in the new IP
	for i := 0; i < bits/8; i++ {
		// Puts a 1 in the new bit since this is the second half
		extraOne := mask[i] ^ n.Mask[i]
		// New IP is the same as old IP with the extra one at the end
		ip[i] = mask[i] & (n.IP[i] | extraOne)
	}

	a = &IPNet{&net.IPNet{IP: n.IP, Mask: mask}}
	b = &IPNet{&net.IPNet{IP: ip, Mask: mask}}
	return
}

// CanCombineWith returns true if the network n can be combined with m
// into one larger cidr twice the size. If true, it returns the combined
// network.
func (n *IPNet) CanCombineWith(m *IPNet) (ok bool, newNet *IPNet) {
	if n.IP.Equal(m.IP) {
		return
	}
	if bytes.Compare(n.Mask, m.Mask) != 0 {
		return
	}
	ones, bits := n.Mask.Size()
	newNet = &IPNet{&net.IPNet{IP: n.IP, Mask: net.CIDRMask(ones-1, bits)}}
	if newNet.ContainsNet(m) {
		ok = true
		return
	}
	return
}

// ipToNet converts the given IP to a /32 or /128 network depending on the type
// of address.
func ipToNet(ip net.IP) *net.IPNet {
	size := 8 * len(ip)
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(size, size)}
}

// incrementIP returns the given IP + 1
func incrementIP(ip net.IP) (result net.IP) {
	result = net.ParseIP("::")
	if len(ip) == 4 {
		result = net.ParseIP("0.0.0.0").To4()
	}

	carry := true
	for i := len(ip) - 1; i >= 0; i-- {
		result[i] = ip[i]
		if carry {
			result[i]++
			if result[i] != 0 {
				carry = false
			}
		}
	}
	return
}

// Expand returns a slice containing all of the IPs in the net up to
// the given limit
func (n *IPNet) Expand(limit int) []net.IP {
	ones, bits := n.Mask.Size()

	size := limit
	max := 1 << 30
	if bits-ones < 30 {
		max = 1 << uint(bits-ones)
	}
	if max < size {
		size = max
	}
	result := make([]net.IP, size)
	next := n.IP
	for i := 0; i < size; i++ {
		result[i] = next[:]
		next = incrementIP(next)
	}
	return result
}
