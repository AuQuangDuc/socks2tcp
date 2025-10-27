package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/things-go/go-socks5"
	"golang.org/x/net/proxy"
)

type Config struct {
	ListenAddr  string
	TargetSocks string
	User        string // local proxy auth user
	Pass        string // local proxy auth pass
	UpUser      string // upstream auth user
	UpPass      string // upstream auth pass
}

// RemoteResolver forces remote DNS resolution by not resolving hostnames locally
type RemoteResolver struct{}

func (r *RemoteResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	// Check if it's already an IP address
	if ip := net.ParseIP(name); ip != nil {
		return ctx, ip, nil
	}

	// For hostnames, don't resolve locally - let the target SOCKS5 proxy handle DNS resolution
	// Return nil IP to indicate remote resolution should be used
	log.Printf("Using remote DNS resolution for hostname: %s", name)
	return ctx, nil, nil
}

// LoggingConn wraps a net.Conn to log bidirectional traffic
type LoggingConn struct {
	net.Conn
	addr         string
	bytesRead    int64
	bytesWritten int64
	mu           sync.Mutex
}

func (lc *LoggingConn) Read(b []byte) (n int, err error) {
	n, err = lc.Conn.Read(b)
	if n > 0 {
		lc.mu.Lock()
		lc.bytesRead += int64(n)
		total := lc.bytesRead
		lc.mu.Unlock()
		log.Printf("← [%s] Read %d bytes (total: %d bytes)", lc.addr, n, total)
	}
	return n, err
}

func (lc *LoggingConn) Write(b []byte) (n int, err error) {
	n, err = lc.Conn.Write(b)
	if n > 0 {
		lc.mu.Lock()
		lc.bytesWritten += int64(n)
		total := lc.bytesWritten
		lc.mu.Unlock()
		log.Printf("→ [%s] Wrote %d bytes (total: %d bytes)", lc.addr, n, total)
	}
	return n, err
}

func (lc *LoggingConn) Close() error {
	lc.mu.Lock()
	totalRead := lc.bytesRead
	totalWritten := lc.bytesWritten
	lc.mu.Unlock()
	log.Printf("✗ [%s] Connection closed - Total read: %d bytes, Total written: %d bytes",
		lc.addr, totalRead, totalWritten)
	return lc.Conn.Close()
}

func main() {
	var config Config

	flag.StringVar(&config.ListenAddr, "l", "", "Listen address for SOCKS5 proxy (e.g., 127.0.0.1:1080)")
	flag.StringVar(&config.TargetSocks, "r", "", "Target SOCKS5 proxy address (e.g., 127.0.0.1:1081)")
	flag.StringVar(&config.User, "user", "", "Local SOCKS5 username for authentication")
	flag.StringVar(&config.Pass, "pass", "", "Local SOCKS5 password for authentication")
	// New flags for upstream auth
	flag.StringVar(&config.UpUser, "ruser", "", "Upstream SOCKS5 username for authentication (optional)")
	flag.StringVar(&config.UpPass, "rpass", "", "Upstream SOCKS5 password for authentication (optional)")
	flag.Parse()

	if config.ListenAddr == "" {
		fmt.Fprintf(os.Stderr, "Error: Listen address (-l) is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if config.TargetSocks == "" {
		fmt.Fprintf(os.Stderr, "Error: Target SOCKS5 proxy address (-r) is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate target address format
	_, _, err := net.SplitHostPort(config.TargetSocks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid target SOCKS5 address format: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Starting authenticated SOCKS5 proxy on %s, forwarding to SOCKS5 proxy %s", config.ListenAddr, config.TargetSocks)

	// Setup authentication if provided (for local proxy)
	var opts []socks5.Option

	if config.User != "" && config.Pass != "" {
		log.Printf("Local authentication enabled for user: %s", config.User)
		// Create credential store
		creds := socks5.StaticCredentials{
			config.User: config.Pass,
		}
		opts = append(opts, socks5.WithCredential(creds))
	} else {
		log.Printf("No local authentication required")
	}

	// Create dialer to target SOCKS5 proxy (allow optional auth to upstream)
	var upAuth *proxy.Auth
	if config.UpUser != "" {
		upAuth = &proxy.Auth{
			User:     config.UpUser,
			Password: config.UpPass,
		}
		log.Printf("Upstream authentication will be used for user: %s", config.UpUser)
	} else {
		log.Printf("No upstream authentication provided; connecting to upstream without auth")
	}

	targetDialer, err := proxy.SOCKS5("tcp", config.TargetSocks, upAuth, proxy.Direct)
	if err != nil {
		log.Fatalf("Failed to create SOCKS5 dialer to target proxy: %v", err)
	}

	// Custom resolver that uses remote DNS resolution
	opts = append(opts, socks5.WithResolver(&RemoteResolver{}))

	// Add custom dialer that forwards through target SOCKS5 proxy with bidirectional traffic logging
	opts = append(opts, socks5.WithDial(func(ctx context.Context, network, addr string) (net.Conn, error) {
		log.Printf("Establishing connection to %s through SOCKS5 proxy %s", addr, config.TargetSocks)
		
		// Connect to target SOCKS5 proxy (this dialer already handles upstream auth if provided)
		targetConn, err := targetDialer.Dial(network, addr)
		if err != nil {
			log.Printf("Failed to connect to %s through SOCKS5 proxy: %v", addr, err)
			return nil, err
		}

		log.Printf("Successfully established bidirectional connection to %s", addr)

		// Wrap the connection to log traffic stats
		return &LoggingConn{
			Conn: targetConn,
			addr: addr,
		}, nil
	}))

	// Add logger
	opts = append(opts, socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))))

	// Create SOCKS5 server
	server := socks5.NewServer(opts...)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping server...")
		os.Exit(0)
	}()

	log.Printf("Authenticated SOCKS5 proxy server started successfully on %s", config.ListenAddr)
	log.Printf("All traffic will be forwarded through SOCKS5 proxy: %s", config.TargetSocks)
	log.Println("Press Ctrl+C to stop the server")

	// Start server
	if err := server.ListenAndServe("tcp", config.ListenAddr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
