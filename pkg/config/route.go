package config

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/kaweezle/kaweezle/pkg/wsl"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const netmask = "255.255.255.255"

func addRoute(destination, mask, gateway string) error {
	cmd := exec.Command("route", "ADD", destination, "MASK", mask, gateway)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add route: %v, output: %s", err, string(output))
	}
	fmt.Printf("Route added successfully: %s", output)
	return nil
}

// TODO: Should return cmd output
func RouteToWSL(distributionName string, fixedAddress string) error {
	wslGateway, err := wsl.FirewallInterface(distributionName)
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
	log.WithFields(fields).Info("Adding route to WSL")
	if admin {
		return addRoute(fixedAddress, netmask, wslGateway)
	} else {
		ctx := context.Background()
		var client ElevatedConfigurationClient
		client, err = GetElevatedClient()
		if err != nil {
			return errors.Wrap(err, "while getting elevated client")
		}
		_, err = client.AddRoute(ctx, &AddRouteRequest{
			FixedAddress: fixedAddress,
			Netmask:      netmask,
			Gateway:      wslGateway,
		})
		return err
	}
}
