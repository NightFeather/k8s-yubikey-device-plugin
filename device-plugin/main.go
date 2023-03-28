package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/google/gousb"
	"github.com/google/gousb/usbid"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"golang.org/x/net/context"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type YubikeyPlugin struct {
  KEYs map[string]map[string]int
  Ctx *gousb.Context
  Heartbeat chan bool
}

func (p *YubikeyPlugin) Start() error {
  p.Ctx = gousb.NewContext()
  return nil
}
func (p *YubikeyPlugin) Stop() error {
  p.Ctx.Close()
  return nil
}

func (p *YubikeyPlugin) GetDevicePluginOptions(ctx context.Context, e *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
  return &pluginapi.DevicePluginOptions{}, nil
}

func (p *YubikeyPlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
  
  devs, err := p.Ctx.OpenDevices(func (desc *gousb.DeviceDesc) bool {
    if desc.Vendor == 0x1050 {
      devs := make([]*pluginapi.DeviceSpec, 1)
      devs[0] = &pluginapi.DeviceSpec{
        HostPath: fmt.Sprintf("/dev/bus/usb/%03d/%03d", desc.Bus, desc.Address),
        ContainerPath: fmt.Sprintf("/dev/bus/usb/%03d/%03d", desc.Bus, desc.Address),,
        Permissions: "rw",
      }
    }
    return false
  });

  if err == nil {
    for _, d := range devs { d.Close() }
  } else {
    log.Default().Println(err)
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
  Heartbeat chan bool
}

func (l *Lister) Discover(pluglistch chan dpm.PluginNameList) {
  for {
    select {
    case <-pluglistch:
      return
    }
  }
}
func (l *Lister) NewPlugin(resourceName string) dpm.PluginInterface { return &YubikeyPlugin { Heartbeat: l.Heartbeat } }
func (l *Lister) GetResourceNamespace() string { return "somewhere.here" }

func ListDevicesAndExit() {
  ctx := gousb.NewContext()
  defer ctx.Close()

  devs, err := ctx.OpenDevices(func (desc *gousb.DeviceDesc) bool {
    match := desc.Vendor == 0x1050
    if !match { return false }
    fmt.Printf("%03d.%03d %s:%s %s\n", desc.Bus, desc.Address, desc.Vendor, desc.Product, usbid.Describe(desc))

    // prevent allocation
    return false
  })

  // cleanup
  defer func() {
    for _, d := range devs { d.Close() }
  }()

  if err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(-1)
  }

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
  manager := dpm.NewManager(&Lister { Heartbeat: make(chan bool) })

  manager.Run()
}
