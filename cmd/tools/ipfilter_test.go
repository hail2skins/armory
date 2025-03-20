package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hail2skins/armory/internal/services/stripe"
)

func main() {
	// Parse command line flags
	ipPtr := flag.String("ip", "", "IP address to check against Stripe IPs")
	refreshPtr := flag.Bool("refresh", false, "Force a refresh of IP ranges")
	monitorPtr := flag.Bool("monitor", false, "Monitor IP ranges in the background")
	statusPtr := flag.Bool("status", false, "Show current status of IP ranges")
	flag.Parse()

	fmt.Println("Stripe IP Filter Test Tool")
	fmt.Println("==========================")

	// Create the IP filter service
	ipFilter := stripe.NewIPFilterService(nil)

	// Handle the status flag
	if *statusPtr {
		status := ipFilter.GetLastUpdateStatus()
		fmt.Println("Current Status:")
		fmt.Println("  Last Update:", status.LastUpdate.Format(time.RFC3339))
		fmt.Println("  Number of IP Ranges:", status.NumRanges)
		fmt.Println("  Last Update Failed:", status.Failed)
		return
	}

	// Handle the refresh flag
	if *refreshPtr {
		fmt.Println("Refreshing IP ranges...")
		err := ipFilter.FetchIPRanges()
		if err != nil {
			log.Fatalf("Failed to refresh IP ranges: %v", err)
		}
		fmt.Println("IP ranges refreshed successfully")

		// Display the status after refresh
		status := ipFilter.GetLastUpdateStatus()
		fmt.Println("Updated Status:")
		fmt.Println("  Number of IP Ranges:", status.NumRanges)
		fmt.Println("  Last Update:", status.LastUpdate.Format(time.RFC3339))
	}

	// Handle the ip flag
	if *ipPtr != "" {
		// If we haven't refreshed, do so now
		if !*refreshPtr {
			status := ipFilter.GetLastUpdateStatus()
			if status.NumRanges == 0 {
				fmt.Println("No IP ranges loaded, refreshing...")
				err := ipFilter.FetchIPRanges()
				if err != nil {
					log.Fatalf("Failed to refresh IP ranges: %v", err)
				}
				fmt.Println("IP ranges refreshed successfully")
			}
		}

		// Check the IP
		allowed := ipFilter.IsStripeIP(*ipPtr)
		if allowed {
			fmt.Printf("✅ IP %s IS recognized as a Stripe IP\n", *ipPtr)
		} else {
			fmt.Printf("❌ IP %s is NOT recognized as a Stripe IP\n", *ipPtr)
		}
	}

	// Handle the monitor flag
	if *monitorPtr {
		fmt.Println("Starting background monitoring of IP ranges...")

		// Create a channel to stop the background refresh
		stop := make(chan struct{})

		// Create a channel to handle OS signals
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		// Start the background refresh
		ipFilter.StartBackgroundRefresh(stop)

		fmt.Println("Press Ctrl+C to stop monitoring")

		// Wait for an OS signal
		<-sigs

		fmt.Println("Stopping background monitoring...")

		// Stop the background refresh
		close(stop)

		// Give it a moment to clean up
		time.Sleep(100 * time.Millisecond)

		fmt.Println("Monitoring stopped")
	}

	// If no flags were provided, print usage
	if !*refreshPtr && *ipPtr == "" && !*monitorPtr && !*statusPtr {
		fmt.Println("Usage:")
		flag.PrintDefaults()
	}
}
