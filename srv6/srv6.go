package srv6

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/kenshaw/baseconv"
	"github.com/vishvananda/netlink"

	"github.com/datum-cloud/galactic-agent/srv6/neighborproxy"
	"github.com/datum-cloud/galactic-agent/srv6/routeegress"
	"github.com/datum-cloud/galactic-agent/srv6/routeingress"
	"github.com/datum-cloud/galactic/util"
)

func RouteIngressAdd(ipStr string) error {
	ip, err := util.ParseIP(ipStr)
	if err != nil {
		log.Fatalf("Invalid ip: %v", err)
	}
	vpc, vpcAttachment, err := util.ExtractVPCFromSRv6Endpoint(ip)
	if err != nil {
		log.Fatalf("could not extract SRv6 endpoint: %v", err)
	}
	vpc, err = ToBase62(vpc)
	if err != nil {
		log.Fatalf("Invalid vpc: %v", err)
	}
	vpcAttachment, err = ToBase62(vpcAttachment)
	if err != nil {
		log.Fatalf("Invalid vpcattachment: %v", err)
	}

	if err := routeingress.Add(netlink.NewIPNet(ip), vpc, vpcAttachment); err != nil {
		log.Fatalf("routeingress add failed: %v", err)
	}
	return nil
}

func RouteIngressDel(ipStr string) error {
	ip, err := util.ParseIP(ipStr)
	if err != nil {
		log.Fatalf("Invalid ip: %v", err)
	}
	vpc, vpcAttachment, err := util.ExtractVPCFromSRv6Endpoint(ip)
	if err != nil {
		log.Fatalf("could not extract SRv6 endpoint: %v", err)
	}
	vpc, err = ToBase62(vpc)
	if err != nil {
		log.Fatalf("Invalid vpc: %v", err)
	}
	vpcAttachment, err = ToBase62(vpcAttachment)
	if err != nil {
		log.Fatalf("Invalid vpcattachment: %v", err)
	}

	if err := routeingress.Delete(netlink.NewIPNet(ip), vpc, vpcAttachment); err != nil {
		log.Fatalf("routeingress delete failed: %v", err)
	}
	return nil
}

func RouteEgressAdd(prefixStr, srcStr string, segmentsStr []string) error {
	prefix, err := netlink.ParseIPNet(prefixStr)
	if err != nil {
		log.Fatalf("Invalid prefix: %v", err)
	}
	src, err := util.ParseIP(srcStr)
	if err != nil {
		log.Fatalf("Invalid src: %v", err)
	}
	segments, err := ParseSegments(segmentsStr)
	if err != nil {
		log.Fatalf("Invalid segments: %v", err)
	}

	vpc, vpcAttachment, err := util.ExtractVPCFromSRv6Endpoint(src)
	if err != nil {
		log.Fatalf("could not extract SRv6 endpoint: %v", err)
	}
	vpc, err = ToBase62(vpc)
	if err != nil {
		log.Fatalf("Invalid vpc: %v", err)
	}
	vpcAttachment, err = ToBase62(vpcAttachment)
	if err != nil {
		log.Fatalf("Invalid vpcattachment: %v", err)
	}

	if IsHost(prefix) {
		if err := neighborproxy.Add(prefix, vpc, vpcAttachment); err != nil {
			log.Fatalf("neighborproxy add failed: %v", err)
		}
	}
	if err := routeegress.Add(vpc, vpcAttachment, prefix, segments); err != nil {
		log.Fatalf("routeegress add failed: %v", err)
	}
	return nil
}

func RouteEgressDel(prefixStr, srcStr string, segmentsStr []string) error {
	prefix, err := netlink.ParseIPNet(prefixStr)
	if err != nil {
		log.Fatalf("Invalid prefix: %v", err)
	}
	src, err := util.ParseIP(srcStr)
	if err != nil {
		log.Fatalf("Invalid src: %v", err)
	}
	segments, err := ParseSegments(segmentsStr)
	if err != nil {
		log.Fatalf("Invalid segments: %v", err)
	}

	vpc, vpcAttachment, err := util.ExtractVPCFromSRv6Endpoint(src)
	if err != nil {
		log.Fatalf("could not extract SRv6 endpoint: %v", err)
	}
	vpc, err = ToBase62(vpc)
	if err != nil {
		log.Fatalf("Invalid vpc: %v", err)
	}
	vpcAttachment, err = ToBase62(vpcAttachment)
	if err != nil {
		log.Fatalf("Invalid vpcattachment: %v", err)
	}

	if IsHost(prefix) {
		if err := neighborproxy.Delete(prefix, vpc, vpcAttachment); err != nil {
			log.Fatalf("neighborproxy delete failed: %v", err)
		}
	}
	if err := routeegress.Delete(vpc, vpcAttachment, prefix, segments); err != nil {
		log.Fatalf("routeegress delete failed: %v", err)
	}
	return nil
}

func ToBase62(value string) (string, error) {
	return baseconv.Convert(strings.ToLower(value), baseconv.DigitsHex, baseconv.Digits62)
}

func IsHost(ipNet *net.IPNet) bool {
	ones, bits := ipNet.Mask.Size()
	// host if mask is full length: /32 for IPv4, /128 for IPv6
	return ones == bits
}

func ParseSegments(input []string) ([]net.IP, error) {
	var segments []net.IP
	for _, ipStr := range input {
		ip, err := util.ParseIP(ipStr)
		if err != nil {
			return nil, fmt.Errorf("could not parse ip (%s): %v", ipStr, err)
		}
		if ip.To4() != nil {
			return nil, fmt.Errorf("not an ipv6 address: %s", ipStr)
		}
		segments = append([]net.IP{ip}, segments...)
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("no segments parsed: %v", input)
	}
	return segments, nil
}
