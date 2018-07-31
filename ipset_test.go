package netaddr

import (
	"math/big"
	"math/rand"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	Eights = net.ParseIP("8.8.8.8").To4()
	Nines  = net.ParseIP("9.9.9.9").To4()

	Ten24, _       = ParseIPNet("10.0.0.0/24")
	TenOne24, _    = ParseIPNet("10.0.1.0/24")
	TenTwo24, _    = ParseIPNet("10.0.2.0/24")
	Ten24128, _    = ParseIPNet("10.0.0.128/25")
	Ten24Router    = net.ParseIP("10.0.0.1").To4()
	Ten24Broadcast = net.ParseIP("10.0.0.255").To4()

	V6Net1, _    = ParseIPNet("2001:db8:1234:abcd::/64")
	V6Net2, _    = ParseIPNet("2001:db8:abcd:1234::/64")
	V6Net1Router = net.ParseIP("2001:db8:1234:abcd::1")

	V6NetSize = big.NewInt(0).Lsh(big.NewInt(1), 64) // 2**64 or 18446744073709551616
)

func TestNetDifference(t *testing.T) {
	diff := Ten24.Difference(Ten24128)

	cidr, _ := ParseIPNet("10.0.0.0/25")
	assert.Equal(t, []*IPNet{cidr}, diff)

	cidr, _ = ParseIPNet("10.0.0.120/29")
	diff = Ten24.Difference(cidr)

	cidr1, _ := ParseIPNet("10.0.0.128/25")
	cidr2, _ := ParseIPNet("10.0.0.0/26")
	cidr3, _ := ParseIPNet("10.0.0.64/27")
	cidr4, _ := ParseIPNet("10.0.0.96/28")
	cidr5, _ := ParseIPNet("10.0.0.112/29")
	assert.Equal(t, []*IPNet{cidr1, cidr2, cidr3, cidr4, cidr5}, diff)
}

func TestIPSetInit(t *testing.T) {
	set := IPSet{}

	assert.Equal(t, big.NewInt(0), set.tree.size())
	assert.Equal(t, []error{}, set.tree.validate())
}

func TestIPSetContains(t *testing.T) {
	set := IPSet{}

	assert.Equal(t, big.NewInt(0), set.tree.size())
	assert.False(t, set.Contains(Eights))
	assert.False(t, set.Contains(Nines))
	assert.Equal(t, []error{}, set.tree.validate())
}

func TestIPSetInsert(t *testing.T) {
	set := IPSet{}

	set.Insert(Nines)
	assert.Equal(t, 1, set.tree.numNodes())
	assert.Equal(t, big.NewInt(1), set.tree.size())
	assert.True(t, set.Contains(Nines))
	assert.False(t, set.Contains(Eights))
	set.Insert(Eights)
	assert.Equal(t, 2, set.tree.numNodes())
	assert.True(t, set.Contains(Eights))
	assert.Equal(t, []error{}, set.tree.validate())
}

func TestIPSetInsertIPNetwork(t *testing.T) {
	set := IPSet{}

	set.InsertIPNet(Ten24)
	assert.Equal(t, 1, set.tree.numNodes())
	assert.Equal(t, big.NewInt(256), set.tree.size())
	assert.True(t, set.ContainsIPNet(Ten24))
	assert.True(t, set.ContainsIPNet(Ten24128))
	assert.False(t, set.Contains(Nines))
	assert.False(t, set.Contains(Eights))
	assert.Equal(t, []error{}, set.tree.validate())
}

func TestIPSetInsertMixed(t *testing.T) {
	set := IPSet{}

	set.InsertIPNet(Ten24)
	assert.Equal(t, 1, set.tree.numNodes())
	set.Insert(Eights)
	set.Insert(Nines)
	set.Insert(Ten24Router)
	assert.Equal(t, 3, set.tree.numNodes())
	assert.Equal(t, big.NewInt(258), set.tree.size())
	assert.True(t, set.ContainsIPNet(Ten24))
	assert.True(t, set.ContainsIPNet(Ten24128))
	assert.True(t, set.Contains(Ten24Router))
	assert.True(t, set.Contains(Eights))
	assert.True(t, set.Contains(Nines))
	assert.Equal(t, []error{}, set.tree.validate())
}

func TestIPSetInsertSequential(t *testing.T) {
	set := IPSet{}

	set.Insert(net.ParseIP("192.168.1.0").To4())
	assert.Equal(t, 1, set.tree.numNodes())
	assert.Equal(t, []error{}, set.tree.validate())
	set.Insert(net.ParseIP("192.168.1.1").To4())
	assert.Equal(t, 1, set.tree.numNodes())
	assert.Equal(t, []error{}, set.tree.validate())
	set.Insert(net.ParseIP("192.168.1.2").To4())
	assert.Equal(t, 2, set.tree.numNodes())
	assert.Equal(t, []error{}, set.tree.validate())
	set.Insert(net.ParseIP("192.168.1.3").To4())
	assert.Equal(t, 1, set.tree.numNodes())
	assert.Equal(t, []error{}, set.tree.validate())
	assert.Equal(t, big.NewInt(4), set.tree.size())

	cidr, _ := ParseIPNet("192.168.1.0/30")
	assert.True(t, set.ContainsIPNet(cidr))

	cidr, _ = ParseIPNet("192.168.1.4/31")
	set.InsertIPNet(cidr)
	assert.Equal(t, 2, set.tree.numNodes())
	assert.True(t, set.ContainsIPNet(cidr))

	cidr, _ = ParseIPNet("192.168.1.6/31")
	set.InsertIPNet(cidr)
	assert.Equal(t, 1, set.tree.numNodes())
	assert.True(t, set.ContainsIPNet(cidr))

	cidr, _ = ParseIPNet("192.168.1.6/31")
	set.InsertIPNet(cidr)
	assert.Equal(t, 1, set.tree.numNodes())
	assert.True(t, set.ContainsIPNet(cidr))

	cidr, _ = ParseIPNet("192.168.0.240/29")
	set.InsertIPNet(cidr)
	assert.Equal(t, 2, set.tree.numNodes())
	assert.True(t, set.ContainsIPNet(cidr))
	assert.Equal(t, []error{}, set.tree.validate())

	cidr, _ = ParseIPNet("192.168.0.248/29")
	set.InsertIPNet(cidr)
	assert.Equal(t, 2, set.tree.numNodes())
	assert.True(t, set.ContainsIPNet(cidr))
	assert.Equal(t, []error{}, set.tree.validate())
}

func TestIPSetRemove(t *testing.T) {
	set := IPSet{}

	set.InsertIPNet(Ten24)
	assert.Equal(t, 1, set.tree.numNodes())
	set.RemoveIPNet(Ten24128)
	assert.Equal(t, 1, set.tree.numNodes())
	assert.Equal(t, big.NewInt(128), set.tree.size())
	assert.False(t, set.ContainsIPNet(Ten24))
	assert.False(t, set.ContainsIPNet(Ten24128))
	cidr, _ := ParseIPNet("10.0.0.0/25")
	assert.True(t, set.ContainsIPNet(cidr))
	assert.Equal(t, []error{}, set.tree.validate())

	set.Remove(Ten24Router)
	assert.Equal(t, big.NewInt(127), set.tree.size())
	assert.Equal(t, 7, set.tree.numNodes())
	assert.Equal(t, []error{}, set.tree.validate())
}

func TestIPSetRemoveNetworkBroadcast(t *testing.T) {
	set := IPSet{}

	set.InsertIPNet(Ten24)
	assert.Equal(t, 1, set.tree.numNodes())
	set.Remove(Ten24.IP)
	set.Remove(Ten24Broadcast)
	assert.Equal(t, big.NewInt(254), set.tree.size())
	assert.Equal(t, 14, set.tree.numNodes())
	assert.False(t, set.ContainsIPNet(Ten24))
	assert.False(t, set.ContainsIPNet(Ten24128))
	assert.False(t, set.Contains(Ten24Broadcast))
	assert.False(t, set.Contains(Ten24.IP))
	assert.Equal(t, []error{}, set.tree.validate())

	cidr, _ := ParseIPNet("10.0.0.128/26")
	assert.True(t, set.ContainsIPNet(cidr))
	assert.True(t, set.Contains(Ten24Router))

	set.Remove(Ten24Router)
	assert.Equal(t, big.NewInt(253), set.tree.size())
	assert.Equal(t, 13, set.tree.numNodes())
}

func TestIPSetRemoveAll(t *testing.T) {
	set := IPSet{}

	set.InsertIPNet(Ten24)
	cidr1, _ := ParseIPNet("192.168.0.0/25")
	set.InsertIPNet(cidr1)
	assert.Equal(t, 2, set.tree.numNodes())

	cidr2, _ := ParseIPNet("0.0.0.0/0")
	set.RemoveIPNet(cidr2)
	assert.Equal(t, 0, set.tree.numNodes())
	assert.False(t, set.ContainsIPNet(Ten24))
	assert.False(t, set.ContainsIPNet(Ten24128))
	assert.False(t, set.ContainsIPNet(cidr1))
	assert.Equal(t, []error{}, set.tree.validate())
}

func TestIPSet_RemoveTop(t *testing.T) {
	testSet := IPSet{}
	ip1 := net.ParseIP("10.0.0.1")
	ip2 := net.ParseIP("10.0.0.2")

	testSet.Insert(ip2) // top
	testSet.Insert(ip1) // inserted at left
	testSet.Remove(ip2) // remove top node

	assert.True(t, testSet.Contains(ip1))
	assert.False(t, testSet.Contains(ip2))
	assert.Nil(t, testSet.tree.next())
	assert.Equal(t, []error{}, testSet.tree.validate())
}

func TestIPSetInsertOverlapping(t *testing.T) {
	set := IPSet{}

	set.InsertIPNet(Ten24128)
	assert.False(t, set.ContainsIPNet(Ten24))
	assert.Equal(t, 1, set.tree.numNodes())
	set.InsertIPNet(Ten24)
	assert.Equal(t, 1, set.tree.numNodes())
	assert.Equal(t, big.NewInt(256), set.tree.size())
	assert.True(t, set.ContainsIPNet(Ten24))
	assert.True(t, set.Contains(Ten24Router))
	assert.False(t, set.Contains(Eights))
	assert.False(t, set.Contains(Nines))
	assert.Equal(t, []error{}, set.tree.validate())
}

func TestIPSetUnion(t *testing.T) {
	set1, set2 := &IPSet{}, &IPSet{}

	set1.InsertIPNet(Ten24)
	cidr, _ := ParseIPNet("192.168.0.248/29")
	set2.InsertIPNet(cidr)

	set := set1.Union(set2)
	assert.True(t, set.ContainsIPNet(Ten24))
	assert.True(t, set.ContainsIPNet(cidr))
	assert.Equal(t, []error{}, set.tree.validate())
}

func TestIPSetDifference(t *testing.T) {
	set1, set2 := &IPSet{}, &IPSet{}

	set1.InsertIPNet(Ten24)
	cidr, _ := ParseIPNet("192.168.0.248/29")
	set2.InsertIPNet(cidr)

	set := set1.Difference(set2)
	assert.True(t, set.ContainsIPNet(Ten24))
	assert.False(t, set.ContainsIPNet(cidr))
	assert.Equal(t, []error{}, set.tree.validate())
}

func TestIntersectionAinB1(t *testing.T) {
	case1 := []string{"10.0.16.0/20", "10.5.8.0/24", "10.23.224.0/23"}
	case2 := []string{"10.0.20.0/30", "10.5.8.0/29", "10.23.224.0/27"}
	output := []string{"10.23.224.0/27", "10.0.20.0/30", "10.5.8.0/29"}
	testIntersection(t, case1, case2, output)

}

func TestIntersectionAinB2(t *testing.T) {
	case1 := []string{"10.10.0.0/30", "10.5.8.0/29", "10.23.224.0/27"}
	case2 := []string{"10.10.0.0/20", "10.5.8.0/24", "10.23.224.0/23"}
	output := []string{"10.10.0.0/30", "10.5.8.0/29", "10.23.224.0/27"}
	testIntersection(t, case1, case2, output)
}

func TestIntersectionAinB3(t *testing.T) {
	case1 := []string{"10.0.5.0/24", "10.5.8.0/29", "10.23.224.0/27"}
	case2 := []string{"10.6.0.0/24", "10.9.9.0/29", "10.23.6.0/23"}
	output := []string{}
	testIntersection(t, case1, case2, output)
}

func TestIntersectionAinB4(t *testing.T) {
	case1 := []string{"10.23.6.0/24", "10.5.8.0/29", "10.23.224.0/27"}
	case2 := []string{"10.6.0.0/24", "10.9.9.0/29", "10.23.6.0/29"}
	output := []string{"10.23.6.0/29"}
	testIntersection(t, case1, case2, output)
}

func TestIntersectionAinB5(t *testing.T) {
	case1 := []string{"2001:db8:0:23::/96", "2001:db8:0:20::/96", "2001:db8:0:15::/96"}
	case2 := []string{"2001:db8:0:23::/64", "2001:db8:0:20::/64", "2001:db8:0:15::/64"}
	output := []string{"2001:db8:0:23::/96", "2001:db8:0:20::/96", "2001:db8:0:15::/96"}
	testIntersection(t, case1, case2, output)
}

func TestIntersectionAinB6(t *testing.T) {
	case1 := []string{"2001:db8:0:23::/64", "2001:db8:0:20::/64", "2001:db8:0:15::/64"}
	case2 := []string{"2001:db8:0:23::/96", "2001:db8:0:20::/96", "2001:db8:0:15::/96"}
	output := []string{"2001:db8:0:15::/96", "2001:db8:0:20::/96", "2001:db8:0:23::/96"}
	testIntersection(t, case1, case2, output)
}

func TestIntersectionAinB7(t *testing.T) {
	case1 := []string{"2001:db8:0:23::/64", "2001:db8:0:20::/64", "2001:db8:0:15::/64"}
	case2 := []string{"2001:db8:0:14::/96", "2001:db8:0:10::/96", "2001:db8:0:8::/96"}
	output := []string{}
	testIntersection(t, case1, case2, output)
}

func TestIntersectionAinB8(t *testing.T) {
	case1 := []string{"2001:db8:0:23::/64", "2001:db8:0:20::/64", "172.16.1.0/24"}
	case2 := []string{"2001:db9:0:14::/96", "2001:db9:0:10::/96", "172.16.1.0/28"}
	output := []string{"172.16.1.0/28"}
	testIntersection(t, case1, case2, output)
}

func TestIntersectionAinB9(t *testing.T) {
	case1 := []string{"10.5.8.0/29"}
	case2 := []string{"10.10.0.0/20", "10.5.8.0/24", "10.23.224.0/23"}
	output := []string{"10.5.8.0/29"}
	testIntersection(t, case1, case2, output)
}

func testIntersection(t *testing.T, input1 []string, input2 []string, output []string) {
	set1, set2, interSect := &IPSet{}, &IPSet{}, &IPSet{}
	for i := 0; i < len(input1); i++ {
		cidr, _ := ParseIPNet(input1[i])
		set1.InsertIPNet(cidr)
	}
	for j := 0; j < len(input2); j++ {
		cidr, _ := ParseIPNet(input2[j])
		set2.InsertIPNet(cidr)
	}
	for k := 0; k < len(output); k++ {
		cidr, _ := ParseIPNet(output[k])
		interSect.InsertIPNet(cidr)
	}
	set := set1.Intersection(set2)
	s1 := set.String()
	intSect := interSect.String()
	if !assert.Equal(t, intSect, s1) {
		t.Logf("\nEXPECTED: %s\nACTUAL: %s\n", intSect, s1)
	}
	assert.Equal(t, []error{}, set.tree.validate())
	assert.Equal(t, []error{}, interSect.tree.validate())

}

func TestIPSetInsertV6(t *testing.T) {
	set := IPSet{}

	set.InsertIPNet(V6Net1)
	assert.Equal(t, 1, set.tree.numNodes())
	set.Insert(V6Net1Router)
	assert.Equal(t, 1, set.tree.numNodes())
	assert.Equal(t, V6NetSize, set.tree.size())
	assert.True(t, set.ContainsIPNet(V6Net1))
	assert.False(t, set.ContainsIPNet(V6Net2))
	assert.False(t, set.Contains(Ten24Router))
	assert.True(t, set.Contains(V6Net1Router))
	assert.False(t, set.Contains(Eights))
	assert.False(t, set.Contains(Nines))

	set.InsertIPNet(V6Net2)
	assert.Equal(t, 2, set.tree.numNodes())
	assert.True(t, set.ContainsIPNet(V6Net1))
	assert.True(t, set.ContainsIPNet(V6Net2))
	assert.Equal(t, big.NewInt(0).Mul(big.NewInt(2), V6NetSize), set.tree.size())
	assert.Equal(t, []error{}, set.tree.validate())
}

func TestIPSetAllocateDeallocate(t *testing.T) {
	rand.Seed(29)

	set := IPSet{}

	bigNet, _ := ParseIPNet("15.1.0.0/16")
	set.InsertIPNet(bigNet)

	ips := set.GetIPs(0)
	assert.Equal(t, 65536, len(ips))
	assert.Equal(t, big.NewInt(65536), set.tree.size())

	allocated := &IPSet{}
	for i := 0; i != 16384; i++ {
		allocated.Insert(ips[rand.Intn(65536)])
	}
	assert.Equal(t, big.NewInt(14500), allocated.tree.size())
	assert.Equal(t, []error{}, allocated.tree.validate())
	ips = allocated.GetIPs(0)
	assert.Equal(t, 14500, len(ips))
	for _, ip := range ips {
		assert.True(t, set.Contains(ip))
	}
	assert.Equal(t, []error{}, set.tree.validate())

	available := set.Difference(allocated)
	assert.Equal(t, big.NewInt(51036), available.tree.size())
	ips = available.GetIPs(0)
	for _, ip := range ips {
		assert.True(t, set.Contains(ip))
		assert.False(t, allocated.Contains(ip))
	}
	assert.Equal(t, 51036, len(ips))
	assert.Equal(t, []error{}, available.tree.validate())
}
