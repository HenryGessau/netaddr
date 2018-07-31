package netaddr

import (
	"net"
)

// IPSet is a set of IP addresses
type IPSet struct {
	tree *ipTree
}

// InsertNet ensures this IPSet has the entire given IP network
func (s *IPSet) InsertNet(net *net.IPNet) {
	s.InsertIPNet(&IPNet{net})
}

// InsertIPNet ensures this IPSet has the entire given IP network
func (s *IPSet) InsertIPNet(net *IPNet) {
	if net == nil {
		return
	}

	newNet := net
	for {
		newNode := &ipTree{net: newNet}
		s.tree = s.tree.insert(newNode)

		if s.tree != newNode && newNode.up == nil {
			break
		}

		// The new node was inserted. See if it can be combined with the previous and/or next ones
		prev := newNode.prev()
		if prev != nil {
			if ok, n := prev.net.CanCombineWith(newNet); ok {
				newNet = n
			}
		}
		next := newNode.next()
		if next != nil {
			if ok, n := newNet.CanCombineWith(next.net); ok {
				newNet = n
			}
		}
		if newNet == newNode.net {
			break
		}
	}
}

// RemoveNet ensures that all of the IPs in the given network are removed from
// the set if present.
func (s *IPSet) RemoveNet(net *net.IPNet) {
	s.RemoveIPNet(&IPNet{net})
}

// RemoveIPNet ensures that all of the IPs in the given network are removed from
// the set if present.
func (s *IPSet) RemoveIPNet(net *IPNet) {
	if net == nil {
		return
	}

	s.tree = s.tree.removeNet(net)
}

// ContainsNet returns true iff this IPSet contains all IPs in the given network
func (s *IPSet) ContainsNet(net *net.IPNet) bool {
	return s.ContainsIPNet(&IPNet{net})
}

// ContainsIPNet returns true iff this IPSet contains all IPs in the given network
func (s *IPSet) ContainsIPNet(net *IPNet) bool {
	if s == nil || net == nil {
		return false
	}
	return s.tree.contains(&ipTree{net: net})
}

// Insert ensures this IPSet has the given IP
func (s *IPSet) Insert(ip net.IP) {
	s.InsertNet(ipToNet(ip))
}

// Remove ensures this IPSet does not contain the given IP
func (s *IPSet) Remove(ip net.IP) {
	s.RemoveNet(ipToNet(ip))
}

// Contains returns true iff this IPSet contains the the given IP address
func (s *IPSet) Contains(ip net.IP) bool {
	return s.ContainsNet(ipToNet(ip))
}

// Union computes the union of this IPSet and another set. It returns the
// result as a new set.
func (s *IPSet) Union(other *IPSet) (newSet *IPSet) {
	newSet = &IPSet{}
	s.tree.walk(func(node *ipTree) {
		newSet.InsertIPNet(node.net)
	})
	other.tree.walk(func(node *ipTree) {
		newSet.InsertIPNet(node.net)
	})
	return
}

// Difference computes the set difference between this IPSet and another one
// It returns the result as a new set.
func (s *IPSet) Difference(other *IPSet) (newSet *IPSet) {
	newSet = &IPSet{}
	s.tree.walk(func(node *ipTree) {
		newSet.InsertIPNet(node.net)
	})
	other.tree.walk(func(node *ipTree) {
		newSet.RemoveIPNet(node.net)
	})
	return
}

// GetIPs retrieves a slice of the first IPs in the set ordered by address up
// to the given limit.
func (s *IPSet) GetIPs(limit int) (ips []net.IP) {
	if limit == 0 {
		limit = int(^uint(0) >> 1) // MaxInt
	}
	for node := s.tree.first(); node != nil; node = node.next() {
		ips = append(ips, node.net.Expand(limit-len(ips))...)
	}
	return
}

// Intersection computes the set intersect between this IPSet and another one
// It returns a new set which is the intersection.
func (s *IPSet) Intersection(set1 *IPSet) (interSect *IPSet) {
	interSect = &IPSet{}
	s.tree.walk(func(node *ipTree) {
		if set1.ContainsIPNet(node.net) {
			interSect.InsertIPNet(node.net)
		}
	})
	set1.tree.walk(func(node *ipTree) {
		if s.ContainsIPNet(node.net) {
			interSect.InsertIPNet(node.net)
		}
	})
	return
}

// String returns a list of IP Networks
func (s *IPSet) String() (str []string) {
	for node := s.tree.first(); node != nil; node = node.next() {
		str = append(str, node.net.IPNet.String())
	}
	return
}
