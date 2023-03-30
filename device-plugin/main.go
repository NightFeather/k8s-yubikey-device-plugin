package main

import (
	"flag"
	"fmt"
	"os"

  "github.com/golang/glog"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"golang.org/x/net/context"
  hid "github.com/sstallion/go-hid"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type YubikeyPlugin struct {
  Heartbeat chan bool
}

func (p *YubikeyPlugin) Start() error {
  hid.Init()
  return nil
}
func (p *YubikeyPlugin) Stop() error {
  hid.Exit()
  return nil
}

func (p *YubikeyPlugin) GetDevicePluginOptions(ctx context.Context, e *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
  return &pluginapi.DevicePluginOptions{}, nil
}

func (p *YubikeyPlugin) ScanDevs() ([]*pluginapi.Device, error) {

  devs := []*pluginapi.Device{}
  err := hid.Enumerate(0x1050, hid.ProductIDAny, func(desc *hid.DeviceInfo) error {
    dev := &pluginapi.Device{ID: desc.SerialNbr, Health: pluginapi.Healthy}
    devs = append(devs, dev)
    return nil
  })

  glog.Infof("Found %d devices.\n", len(devs))

  return devs, err
}

func (p *YubikeyPlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
  devs, err := p.ScanDevs()

  if err == nil {
    glog.Infof("Reports %d devices to Daemon.\n", len(devs))
    s.Send(&pluginapi.ListAndWatchResponse{ Devices: devs })

    for {
      select {
      case <- p.Heartbeat:
        devs, err = p.ScanDevs()
        if err != nil {
          s.Send(&pluginapi.ListAndWatchResponse{})
        } else {
          s.Send(&pluginapi.ListAndWatchResponse{ Devices: devs })
        }
      }
    }
  } else {
    glog.Error(err)
  }

  return nil
}

func (p *YubikeyPlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
  return &pluginapi.AllocateResponse{}, nil
}

func (p *YubikeyPlugin) GetPreferredAllocation(ctx context.Context, r *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
  return &pluginapi.PreferredAllocationResponse{}, nil
}

func (p *YubikeyPlugin) PreStartContainer(ctx context.Context, r *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
  return &pluginapi.PreStartContainerResponse{}, nil
}

type Lister struct {
  UpdateDevChan chan dpm.PluginNameList
  Heartbeat chan bool
}

func (l *Lister) Discover(pluglistch chan dpm.PluginNameList) {
  pluglistch <- []string{"key"}
  for {
    select {
    case devs := <- l.UpdateDevChan:
      pluglistch <- devs
    case <-pluglistch:
      return
    }
  }
}
func (l *Lister) NewPlugin(resourceName string) dpm.PluginInterface {
  glog.Infoln("Try allocating plugin...")
  return &YubikeyPlugin { Heartbeat: l.Heartbeat }
}
func (l *Lister) GetResourceNamespace() string { return "somewhere.here" }

func ListDevicesAndExit() {
  hid.Init()
  defer hid.Exit()

  hid.Enumerate(0x1050, hid.ProductIDAny, func (desc *hid.DeviceInfo) error {
    fmt.Printf("%s %s %s\n",desc.Path, desc.SerialNbr, desc.ProductStr)
    return nil
  })

  os.Exit(0)
}

type Config struct {
  Debug bool
  ListOnly bool
}

func ParseConfig() Config {
  cfg := Config{Debug: false, ListOnly: false}

  flag.Usage = func() {
    flag.PrintDefaults()
  }

  flag.BoolVar(&cfg.ListOnly, "list-devices", false, "show all found devices")
  flag.BoolVar(&cfg.Debug, "debug", false, "set log level to debug")

  flag.Parse()

  return cfg
}

func main() {
  cfg := ParseConfig()

  if cfg.ListOnly { ListDevicesAndExit() }

  l := Lister {
    UpdateDevChan: make(chan dpm.PluginNameList),
    Heartbeat: make(chan bool),
  }

  manager := dpm.NewManager(&l)

  go func() {
    l.Heartbeat <- true
    l.UpdateDevChan <- []string{"key"}
  }()

  manager.Run()
}
