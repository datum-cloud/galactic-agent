package srv6

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/kenshaw/baseconv"
	"github.com/vishvananda/netlink"

	"github.com/datum-cloud/galactic-agent/srv6/neighborproxy"
	"github.com/datum-cloud/galactic-agent/srv6/routeegress"
	"github.com/datum-cloud/galactic-agent/srv6/routeingress"
	"github.com/datum-cloud/galactic-common/util"
)

func RouteIngressAdd(ipStr string) error {
	ip, err := util.ParseIP(ipStr)
	if err != nil {
		return fmt.Errorf("invalid ip: %w", err)
	}
	vpc, vpcAttachment, err := util.ExtractVPCFromSRv6Endpoint(ip)
	if err != nil {
		return fmt.Errorf("could not extract SRv6 endpoint: %w", err)
	}
	vpc, err = ToBase62(vpc)
	if err != nil {
		return fmt.Errorf("invalid vpc: %w", err)
	}
	vpcAttachment, err = ToBase62(vpcAttachment)
	if err != nil {
		return fmt.Errorf("invalid vpcattachment: %w", err)
	}

	if err := routeingress.Add(netlink.NewIPNet(ip), vpc, vpcAttachment); err != nil {
		return fmt.Errorf("routeingress add failed: %w", err)
	}
	return nil
}

func RouteIngressDel(ipStr string) error {
	ip, err := util.ParseIP(ipStr)
	if err != nil {
		return fmt.Errorf("invalid ip: %w", err)
	}
	vpc, vpcAttachment, err := util.ExtractVPCFromSRv6Endpoint(ip)
	if err != nil {
		return fmt.Errorf("could not extract SRv6 endpoint: %w", err)
	}
	vpc, err = ToBase62(vpc)
	if err != nil {
		return fmt.Errorf("invalid vpc: %w", err)
	}
	vpcAttachment, err = ToBase62(vpcAttachment)
	if err != nil {
		return fmt.Errorf("invalid vpcattachment: %w", err)
	}

	if err := routeingress.Delete(netlink.NewIPNet(ip), vpc, vpcAttachment); err != nil {
		return fmt.Errorf("routeingress delete failed: %w", err)
	}
	return nil
}

func RouteEgressAdd(prefixStr, srcStr string, segmentsStr []string) error {
	prefix, err := netlink.ParseIPNet(prefixStr)
	if err != nil {
		return fmt.Errorf("invalid prefix: %w", err)
	}
	src, err := util.ParseIP(srcStr)
	if err != nil {
		return fmt.Errorf("invalid src: %w", err)
	}
	segments, err := ParseSegments(segmentsStr)
	if err != nil {
		return fmt.Errorf("invalid segments: %w", err)
	}

	vpc, vpcAttachment, err := util.ExtractVPCFromSRv6Endpoint(src)
	if err != nil {
		return fmt.Errorf("could not extract SRv6 endpoint: %w", err)
	}
	vpc, err = ToBase62(vpc)
	if err != nil {
		return fmt.Errorf("invalid vpc: %w", err)
	}
	vpcAttachment, err = ToBase62(vpcAttachment)
	if err != nil {
		return fmt.Errorf("invalid vpcattachment: %w", err)
	}

	var errs []error
	if IsHost(prefix) {
		if err := neighborproxy.Add(prefix, vpc, vpcAttachment); err != nil {
			errs = append(errs, fmt.Errorf("neighborproxy add failed: %w", err))
		}
	}
	if err := routeegress.Add(vpc, vpcAttachment, prefix, segments); err != nil {
		errs = append(errs, fmt.Errorf("routeegress add failed: %w", err))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func RouteEgressDel(prefixStr, srcStr string, segmentsStr []string) error {
	prefix, err := netlink.ParseIPNet(prefixStr)
	if err != nil {
		return fmt.Errorf("invalid prefix: %w", err)
	}
	src, err := util.ParseIP(srcStr)
	if err != nil {
		return fmt.Errorf("invalid src: %w", err)
	}
	segments, err := ParseSegments(segmentsStr)
	if err != nil {
		return fmt.Errorf("invalid segments: %w", err)
	}

	vpc, vpcAttachment, err := util.ExtractVPCFromSRv6Endpoint(src)
	if err != nil {
		return fmt.Errorf("could not extract SRv6 endpoint: %w", err)
	}
	vpc, err = ToBase62(vpc)
	if err != nil {
		return fmt.Errorf("invalid vpc: %w", err)
	}
	vpcAttachment, err = ToBase62(vpcAttachment)
	if err != nil {
		return fmt.Errorf("invalid vpcattachment: %w", err)
	}

	var errs []error
	if IsHost(prefix) {
		if err := neighborproxy.Delete(prefix, vpc, vpcAttachment); err != nil {
			errs = append(errs, fmt.Errorf("neighborproxy delete failed: %w", err))
		}
	}
	if err := routeegress.Delete(vpc, vpcAttachment, prefix, segments); err != nil {
		errs = append(errs, fmt.Errorf("routeegress delete failed: %w", err))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
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
