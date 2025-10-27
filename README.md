# SOCKS2TCP - SOCKS5 to TCP Forwarder

A tool that creates a SOCKS5 proxy with optional authentication that forwards all traffic to a single TCP destination. This is useful for tunneling all SOCKS5 connections through a specific endpoint.

## Features

- **SOCKS5 Proxy**: Full SOCKS5 protocol support with bidirectional traffic forwarding
- **Authentication**: Optional username/password authentication
- **Traffic Forwarding**: All SOCKS5 connections are transparently forwarded to a single TCP destination
- **Logging**: Detailed logging of connections and forwarding activity
- **Graceful Shutdown**: Handles SIGINT/SIGTERM for clean shutdown

## Installation

```bash
go build -o socks2tcp main.go
```

## Usage

```bash
./socks2tcp -l <listen_sock_addr:port> -r <target_tcp_addr> [-user <username>] [-pass <password>]
```

### Command Line Options

- `-l string`: Listen address for SOCKS5 proxy (e.g., 127.0.0.1:1080) **[Required]**
- `-r string`: Target TCP address to forward to (e.g., 127.0.0.1:8080) **[Required]**
- `-user string`: SOCKS5 username for authentication **[Optional]**
- `-pass string`: SOCKS5 password for authentication **[Optional]**
- `-ruser string`: Upstream SOCKS5 username for authentication
- `-rpass string`: Upstream SOCKS5 password for authentication

### Examples

#### Basic usage without authentication:
```bash
./socks2tcp -l 127.0.0.1:1080 -r 192.168.1.100:8080
```

#### With authentication:
```bash
./socks2tcp -l 127.0.0.1:1080 -r 192.168.1.100:8080 -user myuser -pass mypass
```

#### Listen on all interfaces:
```bash
./socks2tcp -l 0.0.0.0:1080 -r 10.0.0.1:443
```

## How It Works

1. **SOCKS5 Server**: The tool creates a SOCKS5 proxy server listening on the specified address
2. **Custom Resolver**: All hostname resolution requests are redirected to resolve to the target IP
3. **Custom Dialer**: All connection attempts are intercepted and redirected to the target TCP address
4. **Bidirectional Forwarding**: Traffic flows in both directions between the SOCKS5 client and the target TCP server

## Use Cases

- **Protocol Tunneling**: Tunnel various protocols through a specific TCP endpoint
- **Traffic Aggregation**: Force all SOCKS5 traffic through a single destination
- **Testing and Development**: Redirect application traffic to test servers
- **Network Pivoting**: Route traffic through specific network endpoints

## Security Considerations

- Use authentication (`-user` and `-pass`) when deploying in untrusted environments
- Consider using TLS/SSL termination at the target endpoint for encryption
- Monitor logs for unauthorized access attempts
- Restrict listening interface (`-l`) to trusted networks when possible

## Dependencies

- [github.com/things-go/go-socks5](https://github.com/things-go/go-socks5) - SOCKS5 protocol implementation

## License

This tool is provided as-is for educational and testing purposes.
