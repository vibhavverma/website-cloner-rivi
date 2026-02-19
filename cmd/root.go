package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	relversion "sigs.k8s.io/release-utils/version"
)

var (
	Open         bool
	Serve        bool
	ServePort    int
	Depth        int
	Parallel     int
	Delay        int
	Headless     bool
	NoHeadless   bool
	WaitFor      string
	WaitTimeout  int
	UserAgent    string
	ProxyString  string
	Cookies      []string

	// Root cmd
	rootCmd = &cobra.Command{
		Use:   "goclone <url>",
		Short: "Clone a website with ease!",
		Long:  `Copy websites to your computer! goclone is a utility that allows you to download a website from the Internet to a local directory. Get html, css, js, images, and other files from the server to your computer. goclone arranges the original site's relative link-structure. Simply open a page of the "mirrored" website in your browser, and you can browse the site from link to link, as if you were viewing it online.`,
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				if err := cmd.Usage(); err != nil {
					log.Fatal(err)
				}
				return
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			opts := CloneOptions{
				Serve:       Serve,
				Open:        Open,
				ServePort:   ServePort,
				Depth:       Depth,
				Parallel:    Parallel,
				Delay:       Delay,
				Headless:    Headless,
				NoHeadless:  NoHeadless,
				WaitFor:     WaitFor,
				WaitTimeout: time.Duration(WaitTimeout) * time.Second,
				Cookies:     Cookies,
				Proxy:       ProxyString,
				UserAgent:   UserAgent,
			}

			if err := CloneSite(ctx, args, opts); err != nil {
				log.Fatalf("%+v", err)
			}
		},
	}
)

// Execute the clone command
func Execute() {
	rootCmd.AddCommand(relversion.Version())

	pf := rootCmd.PersistentFlags()
	pf.BoolVarP(&Open, "open", "o", false, "Automatically open project in default browser")
	pf.BoolVarP(&Serve, "serve", "s", false, "Serve the generated files using Echo.")
	pf.IntVarP(&ServePort, "servePort", "P", 5000, "Serve port number.")
	pf.StringVarP(&ProxyString, "proxy_string", "p", "", "Proxy connection string. Support http and socks5")
	pf.StringVarP(&UserAgent, "user_agent", "u", "", "Custom User Agent")
	pf.IntVarP(&Depth, "depth", "d", 0, "Maximum crawl depth (0 = single page, 1 = page + links, etc.)")
	pf.BoolVar(&Headless, "headless", false, "Force headless browser mode (captures 3D/WebGL assets)")
	pf.BoolVar(&NoHeadless, "no-headless", false, "Disable headless browser even for detected SPAs")
	pf.StringVar(&WaitFor, "wait-for", "", "CSS selector to wait for before capturing (headless mode)")
	pf.IntVar(&WaitTimeout, "wait-timeout", 30, "Maximum seconds to wait for page load (headless mode)")
	pf.IntVar(&Parallel, "parallel", 5, "Maximum parallel downloads")
	pf.IntVar(&Delay, "delay", 100, "Delay between requests in milliseconds")
	rootCmd.Flags().StringSliceVarP(&Cookies, "cookie", "C", nil, "Pre-set these cookies")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
