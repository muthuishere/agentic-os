package groups

import (
	"github.com/muthuishere/agentic-os/internal/cli"
)

// netState is the connectivity snapshot the network group renders.
type netState struct {
	Interface string
	IP        string
	SSID      string // empty when not on Wi-Fi
	Signal    string // free-form, platform-dependent
}

func init() {
	register(func(r *cli.Registry) {
		r.Describe("network", "Connectivity, addresses, and Wi-Fi")
		r.Add(
			&cli.Command{
				Group: "network", Name: "status",
				Summary:  "Print the active interface, address, and Wi-Fi network",
				Examples: []string{"aos network status"},
				Run:      runNetworkStatus,
			},
			&cli.Command{
				Group: "network", Name: "ip",
				Summary: "Print the local address of the active interface",
				Run: func(c *cli.Ctx, _ []string) error {
					state, err := readNetwork()
					if err != nil {
						return err
					}
					if state.IP == "" {
						return &cli.ExitError{Code: 1, Message: "no active address"}
					}
					c.Println(state.IP)
					return nil
				},
			},
			&cli.Command{
				Group: "network", Name: "wifi",
				Summary: "Print the Wi-Fi network currently joined",
				Run: func(c *cli.Ctx, _ []string) error {
					state, err := readNetwork()
					if err != nil {
						return err
					}
					if state.SSID == "" {
						return &cli.ExitError{Code: 1, Message: "not connected to Wi-Fi"}
					}
					if state.Signal != "" {
						c.Printf("%s  %s\n", state.SSID, state.Signal)
					} else {
						c.Println(state.SSID)
					}
					return nil
				},
			},
		)
	})
}

func runNetworkStatus(c *cli.Ctx, _ []string) error {
	state, err := readNetwork()
	if err != nil {
		return err
	}
	if state.Interface == "" && state.IP == "" {
		c.Println("offline")
		return &cli.ExitError{Code: 1}
	}
	c.Printf("interface  %s\n", orDash(state.Interface))
	c.Printf("address    %s\n", orDash(state.IP))
	if state.SSID != "" {
		c.Printf("wifi       %s\n", state.SSID)
		if state.Signal != "" {
			c.Printf("signal     %s\n", state.Signal)
		}
	}
	return nil
}

func orDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
