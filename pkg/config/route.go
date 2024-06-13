package config

import (
	"context"
	"fmt"
	"net"
	"os/exec"

	"github.com/kaweezle/kaweezle/pkg/wsl"
	netroute "github.com/libp2p/go-netroute"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const netmask = "255.255.255.255"

func addRoute(destination, mask, gateway string) error {
	cmd := exec.Command("route", "-P", "ADD", destination, "MASK", mask, gateway)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add route: %v, output: %s", err, string(output))
	}
	fmt.Printf("Route added successfully: %s", output)
	return nil
}

func removeRoute(destination string) error {
	cmd := exec.Command("route", "DELETE", destination)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove route: %v, output: %s", err, string(output))
	}
	fmt.Printf("Route removed successfully: %s", output)
	return nil
}

func RouteToWSL(distributionName string, fixedAddress string, remove bool) error {
	wslGateway, err := wsl.GetNatGatewayIpAddress()
	if err != nil {
		return errors.Wrap(err, "failed to get WSL gateway")
	}
	admin := IsAdmin()
	fields := log.Fields{
		"distribution": distributionName,
		"fixedAddress": fixedAddress,
		"wslGateway":   wslGateway,
		"admin":        admin,
	}
	fixedAddressIP := net.ParseIP(fixedAddress)
	if fixedAddressIP == nil {
		return errors.Errorf("failed to parse WSL gateway IP: %s", wslGateway)
	}

	r, err := netroute.New()
	if err != nil {
		return errors.Wrap(err, "while creating netroute")
	}
	iface, _, src, err := r.Route(fixedAddressIP)
	routed := err == nil && src.String() == wslGateway
	if routed && !remove {
		log.WithFields(fields).Infof("Route already exists to %s via %s on %s", fixedAddress, src, iface.Name)
		return nil
	}
	if !routed && remove {
		log.WithFields(fields).Infof("Route does not exist to %s via %s", fixedAddress, wslGateway)
		return nil
	}

	log.WithFields(fields).Info("Adding route to WSL")
	if admin {
		if remove {
			return removeRoute(fixedAddress)
		} else {
			return addRoute(fixedAddress, netmask, wslGateway)
		}
	} else {
		ctx := context.Background()
		var client ElevatedConfigurationClient
		client, err = GetElevatedClient()
		if err != nil {
			return errors.Wrap(err, "while getting elevated client")
		}
		if remove {
			_, err = client.RemoveRoute(ctx, &RemoveRouteRequest{
				FixedAddress: fixedAddress,
			})
		} else {
			_, err = client.AddRoute(ctx, &AddRouteRequest{
				FixedAddress: fixedAddress,
				Netmask:      netmask,
				Gateway:      wslGateway,
			})

		}
		return err
	}
}
