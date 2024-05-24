package config

import (
	context "context"
	"net"
	"os"
	"strings"

	"fmt"
	"math/rand"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/kaweezle/kaweezle/pkg/logger"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	status "google.golang.org/grpc/status"
)

type ElevatedConfigurationServerImpl struct {
	UnimplementedElevatedConfigurationServer
	Done chan bool
}

func (ec ElevatedConfigurationServerImpl) AddRoute(ctx context.Context, r *AddRouteRequest) (*AddRouteResponse, error) {
	err := addRoute(r.FixedAddress, r.Netmask, r.Gateway)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	return &AddRouteResponse{}, nil

}
func (ec ElevatedConfigurationServerImpl) ConfigureDomains(ctx context.Context, r *ConfigureDomainsRequest) (*ConfigureDomainsResponse, error) {
	logrus.WithFields(logrus.Fields{
		"distribution_name": r.DistributionName,
		"ip_address":        r.IpAddress,
		"domains":           strings.Join(r.Domains, " "),
		"remove":            r.Remove,
	}).Info("Received domain request")
	result, err := ConfigureDomains(r.DistributionName, r.IpAddress, r.Domains, r.Remove)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	logrus.WithField("domains", strings.Join(result, " ")).Info("Returning...")
	response := &ConfigureDomainsResponse{Domains: result}
	return response, nil
}

func (ec ElevatedConfigurationServerImpl) Stop(ctx context.Context, r *StopRequest) (*StopResponse, error) {

	logrus.Info("Asked to stop...")
	defer func() {
		logrus.Info("Sending stop signal...")
		ec.Done <- true
	}()
	return &StopResponse{}, nil
}

func generateRandomPipeName() string {
	rand.Seed(time.Now().UnixNano())
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return fmt.Sprintf("\\\\.\\pipe\\%s", string(b))
}

var startElevatedFields = logrus.Fields{
	logger.TaskKey: "Start Elevated server",
}

func IsAdmin() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

const (
	// SEE_MASK_NO_CONSOLE: Do not display a console window
	SEE_MASK_NO_CONSOLE = 0x00008000
	SW_SHOW             = 5
	SW_HIDE             = 0
)

func StartElevatedServer() (ElevatedConfigurationClient, error) {
	pipeName := generateRandomPipeName()

	verbPtr, _ := windows.UTF16PtrFromString("runas")
	cmdName, _ := os.Executable()
	cmdPtr, _ := windows.UTF16PtrFromString(cmdName)
	argsPtr, _ := windows.UTF16PtrFromString(strings.Join([]string{"configure", "elevate", pipeName}, " "))
	cwd, _ := os.Getwd()
	cwdPtr, _ := windows.UTF16PtrFromString(cwd)

	logrus.WithFields(startElevatedFields).WithField("pipe_name", pipeName).Info("Starting elevated server...")
	if err := windows.ShellExecute(0, verbPtr, cmdPtr, argsPtr, cwdPtr, SW_HIDE); err != nil {
		return nil, errors.Wrap(err, "failed to run elevated server")
	}

	// now create the client
	grpcClient, err := grpc.NewClient("localhost:50005",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			time.Sleep(2 * time.Second)

			for {
				conn, err := winio.DialPipe(pipeName, nil)
				if err == nil {
					logrus.WithFields(startElevatedFields).WithField("pipe_name", pipeName).Info("Connected to elevated server.")
					return conn, err
				}
				time.Sleep(200 * time.Millisecond)
			}
		}))
	// grpcClient, err := grpc.NewClient("localhost:50005", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrapf(err, "while creating client")
	}
	grpcClient.Connect()

	client := NewElevatedConfigurationClient(grpcClient)

	return client, nil
}

var elevatedClient ElevatedConfigurationClient

func GetElevatedClient() (ElevatedConfigurationClient, error) {
	var err error
	if elevatedClient == nil {
		elevatedClient, err = StartElevatedServer()
		if err != nil {
			return nil, errors.Wrapf(err, "error while starting elevated server")
		}
	}
	return elevatedClient, err
}

func ReleaseElevatedClient(ctx context.Context) error {
	if elevatedClient != nil {
		_, err := elevatedClient.Stop(ctx, &StopRequest{})
		logrus.WithError(err).Info("Stopped elevated server.")
		return err
	}
	return nil
}
