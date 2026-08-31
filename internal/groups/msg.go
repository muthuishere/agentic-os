package groups

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/muthuishere/agentic-os/internal/cli"
	"github.com/muthuishere/agentic-os/internal/msg"
)

func init() {
	register(func(r *cli.Registry) {
		r.Describe("msg", "Send and receive messages through the local messenger hub")
		r.Add(
			&cli.Command{
				Group: "msg", Name: "health",
				Summary:  "Report whether the messenger hub is up, and its channels",
				Examples: []string{"aos msg health"},
				Run:      runMsgHealth,
			},
			&cli.Command{
				Group: "msg", Name: "send",
				Summary: "Send a message on a channel",
				Args:    "--channel=<name> <text...> [--to=<thread>] [--reply-to=<id|last>] [--file=<path>]",
				Examples: []string{
					`aos msg send --channel=ops "build passed"`,
					`aos msg send --channel=ops "on it" --reply-to=last`,
					`aos msg send --channel=ops --file=report.pdf "this month"`,
				},
				Run: runMsgSend,
			},
			&cli.Command{
				Group: "msg", Name: "inbox",
				Summary: "Poll the hub once and print new envelopes",
				Args:    "[--since=<cursor>] [--channel=<name>]",
				Examples: []string{
					"aos msg inbox",
					"aos msg inbox --since=7",
				},
				Run: runMsgInbox,
			},
			&cli.Command{
				Group: "msg", Name: "listen",
				Summary:  "Follow the hub, printing each envelope as one JSON line",
				Blocking: true,
				Args:     "[--since=<cursor>] [--channel=<name>] [--interval=<ms>]",
				Examples: []string{
					"aos msg listen",
					"aos msg listen --channel=ops --interval=1000",
				},
				Run: runMsgListen,
			},
		)
	})
}

func runMsgHealth(c *cli.Ctx, _ []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := msg.New(c.Env)
	health, err := client.Health(ctx)
	if err != nil {
		return &cli.ExitError{Code: 1, Message: err.Error()}
	}
	c.Printf("hub       %s\n", client.BaseURL)
	c.Printf("service   %s\n", health.Service)
	if len(health.Channels) == 0 {
		c.Println("channels  none configured")
		return nil
	}
	width := 0
	for name := range health.Channels {
		if len(name) > width {
			width = len(name)
		}
	}
	c.Println("channels")
	for name, kind := range health.Channels {
		c.Printf("  %-*s  %s\n", width, name, kind)
	}
	return nil
}

func runMsgSend(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args, "channel", "to", "reply-to", "file")
	if err != nil {
		return err
	}
	if err := set.Reject("channel", "to", "reply-to", "file"); err != nil {
		return err
	}

	req := msg.SendRequest{
		Channel: set.String("channel", ""),
		Text:    strings.Join(set.Rest, " "),
		To:      set.String("to", ""),
		ReplyTo: set.String("reply-to", ""),
		File:    set.String("file", ""),
	}
	if req.Channel == "" {
		return fmt.Errorf("`msg send` needs --channel=<name>; `msg health` lists them")
	}
	if req.Text == "" && req.File == "" {
		return fmt.Errorf("`msg send` needs text, a --file, or both")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	res, err := msg.New(c.Env).Send(ctx, req)
	if err != nil {
		return &cli.ExitError{Code: 1, Message: err.Error()}
	}
	// The provider id is what threads a later reply, so it is the useful output.
	c.Println(res.ID)
	return nil
}

func runMsgInbox(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args, "since", "channel")
	if err != nil {
		return err
	}
	if err := set.Reject("since", "channel"); err != nil {
		return err
	}
	since, err := set.Int("since", 0)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inbox, err := msg.New(c.Env).Inbox(ctx, int64(since))
	if err != nil {
		return &cli.ExitError{Code: 1, Message: err.Error()}
	}
	printed := printEnvelopes(c, inbox.Messages, set.String("channel", ""))
	// The cursor goes to stderr so stdout stays a clean stream of envelopes.
	c.Warnf("next=%d printed=%d\n", inbox.Next, printed)
	return nil
}

func runMsgListen(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args, "since", "channel", "interval")
	if err != nil {
		return err
	}
	if err := set.Reject("since", "channel", "interval"); err != nil {
		return err
	}
	since, err := set.Int("since", 0)
	if err != nil {
		return err
	}
	interval, err := set.Int("interval", 2000)
	if err != nil {
		return err
	}
	if interval < 200 {
		return fmt.Errorf("--interval must be at least 200ms")
	}
	channel := set.String("channel", "")

	// Ctrl-C should end the loop cleanly rather than killing it mid-write.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	client := msg.New(c.Env)
	cursor := int64(since)
	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	defer ticker.Stop()

	for {
		inbox, err := client.Inbox(ctx, cursor)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			// A hub restart should not end a long-running listener; report and
			// keep trying on the next tick.
			c.Warnf("aos: %v\n", err)
		} else {
			printEnvelopes(c, inbox.Messages, channel)
			cursor = inbox.Next
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// printEnvelopes writes one envelope per line as compact JSON, optionally
// filtered to a single channel, and returns how many it printed.
func printEnvelopes(c *cli.Ctx, envelopes []json.RawMessage, channel string) int {
	printed := 0
	for _, envelope := range envelopes {
		if channel != "" && envelopeChannel(envelope) != channel {
			continue
		}
		c.Printf("%s\n", strings.TrimSpace(string(envelope)))
		printed++
	}
	return printed
}

func envelopeChannel(envelope json.RawMessage) string {
	var peek struct {
		Channel string `json:"channel"`
	}
	if err := json.Unmarshal(envelope, &peek); err != nil {
		return ""
	}
	return peek.Channel
}
