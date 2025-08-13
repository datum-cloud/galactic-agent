package srv6

import (
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

func RouteIngressAdd(ipRaw string) error {
	ip, err := netlink.ParseIPNet(ipRaw)
	if err != nil {
		log.Fatalf("Invalid ip: %v", err)
	}
	if !IsHost(ip) {
		log.Fatalf("ip is not a host route")
	}
	vpc, vpcAttachment, err := util.ExtractVPCFromSRv6Endpoint(ip.IP)
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

	if err := routeingress.Add(ip, vpc, vpcAttachment); err != nil {
		log.Fatalf("routeingress add failed: %v", err)
	}
	return nil
}

func RouteIngressDel(ipRaw string) error {
	ip, err := netlink.ParseIPNet(ipRaw)
	if err != nil {
		log.Fatalf("Invalid ip: %v", err)
	}
	if !IsHost(ip) {
		log.Fatalf("ip is not a host route")
	}
	vpc, vpcAttachment, err := util.ExtractVPCFromSRv6Endpoint(ip.IP)
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

	if err := routeingress.Delete(ip, vpc, vpcAttachment); err != nil {
		log.Fatalf("routeingress delete failed: %v", err)
	}
	return nil
}

func RouteEgressAdd(prefixRaw, srcRaw, segmentsRaw string) error {
	prefix, err := netlink.ParseIPNet(prefixRaw)
	if err != nil {
		log.Fatalf("Invalid prefix: %v", err)
	}
	src, err := netlink.ParseIPNet(srcRaw)
	if err != nil {
		log.Fatalf("Invalid src: %v", err)
	}
	if !IsHost(src) {
		log.Fatalf("src is not a host route")
	}
	segments, err := util.ParseSegments(segmentsRaw)
	if err != nil {
		log.Fatalf("Invalid segments: %v", err)
	}

	vpc, vpcAttachment, err := util.ExtractVPCFromSRv6Endpoint(src.IP)
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

func RouteEgressDel(prefixRaw, srcRaw, segmentsRaw string) error {
	prefix, err := netlink.ParseIPNet(prefixRaw)
	if err != nil {
		log.Fatalf("Invalid prefix: %v", err)
	}
	src, err := netlink.ParseIPNet(srcRaw)
	if err != nil {
		log.Fatalf("Invalid src: %v", err)
	}
	if !IsHost(src) {
		log.Fatalf("src is not a host route")
	}
	segments, err := util.ParseSegments(segmentsRaw)
	if err != nil {
		log.Fatalf("Invalid segments: %v", err)
	}

	vpc, vpcAttachment, err := util.ExtractVPCFromSRv6Endpoint(src.IP)
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
