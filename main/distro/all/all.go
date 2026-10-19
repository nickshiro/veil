package all

import (
	// The following are necessary as they register handlers in their init functions.

	// Mandatory features. Can't remove unless there are replacements.
	_ "github.com/nickshiro/veil/app/dispatcher"
	_ "github.com/nickshiro/veil/app/proxyman/inbound"
	_ "github.com/nickshiro/veil/app/proxyman/outbound"

	// Default commander and all its services. This is an optional feature.
	_ "github.com/nickshiro/veil/app/commander"
	_ "github.com/nickshiro/veil/app/log/command"
	_ "github.com/nickshiro/veil/app/proxyman/command"
	_ "github.com/nickshiro/veil/app/stats/command"

	// Developer preview services
	_ "github.com/nickshiro/veil/app/observatory/command"

	// Other optional features.
	_ "github.com/nickshiro/veil/app/dns"
	_ "github.com/nickshiro/veil/app/dns/fakedns"
	_ "github.com/nickshiro/veil/app/geodata"
	_ "github.com/nickshiro/veil/app/log"
	_ "github.com/nickshiro/veil/app/metrics"
	_ "github.com/nickshiro/veil/app/policy"
	_ "github.com/nickshiro/veil/app/reverse"
	_ "github.com/nickshiro/veil/app/router"
	_ "github.com/nickshiro/veil/app/stats"

	// Fix dependency cycle caused by core import in internet package
	_ "github.com/nickshiro/veil/transport/internet/tagged/taggedimpl"

	// Developer preview features
	_ "github.com/nickshiro/veil/app/observatory"

	// Inbound and outbound proxies.
	_ "github.com/nickshiro/veil/proxy/blackhole"
	_ "github.com/nickshiro/veil/proxy/dns"
	_ "github.com/nickshiro/veil/proxy/dokodemo"
	_ "github.com/nickshiro/veil/proxy/freedom"
	_ "github.com/nickshiro/veil/proxy/http"
	_ "github.com/nickshiro/veil/proxy/loopback"
	_ "github.com/nickshiro/veil/proxy/masque"
	_ "github.com/nickshiro/veil/proxy/shadowsocks"
	_ "github.com/nickshiro/veil/proxy/socks"
	_ "github.com/nickshiro/veil/proxy/trojan"
	_ "github.com/nickshiro/veil/proxy/vless/inbound"
	_ "github.com/nickshiro/veil/proxy/vless/outbound"
	_ "github.com/nickshiro/veil/proxy/vmess/inbound"
	_ "github.com/nickshiro/veil/proxy/vmess/outbound"
	_ "github.com/nickshiro/veil/proxy/wireguard"

	// Transports
	_ "github.com/nickshiro/veil/transport/internet/grpc"
	_ "github.com/nickshiro/veil/transport/internet/httpupgrade"
	_ "github.com/nickshiro/veil/transport/internet/kcp"
	_ "github.com/nickshiro/veil/transport/internet/masque"
	_ "github.com/nickshiro/veil/transport/internet/reality"
	_ "github.com/nickshiro/veil/transport/internet/splithttp"
	_ "github.com/nickshiro/veil/transport/internet/tcp"
	_ "github.com/nickshiro/veil/transport/internet/tls"
	_ "github.com/nickshiro/veil/transport/internet/udp"
	_ "github.com/nickshiro/veil/transport/internet/websocket"
	_ "github.com/nickshiro/veil/transport/internet/xdrive"

	// Transport headers
	_ "github.com/nickshiro/veil/transport/internet/headers/http"
	_ "github.com/nickshiro/veil/transport/internet/headers/noop"

	// JSON
	_ "github.com/nickshiro/veil/main/json"

	// Load config from file or http(s)
	_ "github.com/nickshiro/veil/main/confloader/external"

	// Commands
	_ "github.com/nickshiro/veil/main/commands/all"
)
