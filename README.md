# Scanblocker

## What

Scanblocker is meant to run on Linux hosts as a service ("daemon") and does the following:  

1. **Network device sniffing.** Listens on a network interface for traffic and TCP connection attempts, i. e., filtering for incoming (outside the host) TCP headers with the SYN flag on (put not the SYN-ACK).
2. **External connection detection.** Prints to `stdout` when such filtered packet is detected, for example `2022-03-23 23:56:06: New connection: 10.128.0.2:40772 -> 10.128.0.3:28214`
3. **External port scan detection.** When more than a number of connection attempts (set at three) are detected from an external source IP to different local host destination ports in the last specific time interval (set to 60 seconds), that's detected as a port scan (multiple connections to the same host port can naturally be legit traffic to a local service). This event is printed out for example `2022-03-23 23:56:06: Port scan detected: 10.128.0.2 -> 10.128.0.3`
4. **Iptables port scan blocking.** When the port scan as defined above is detected, it will try to append an `iptables` rule blocking the source IP of the scan, for example; `iptables -A INPUT -s 10.128.0.2 -j DROP`. If `iptables` is not installed or if for other reason the command fails, the program doesn't exit. Since the packet capture takes place at a lower level than netfilter (iptables) in the Linux stack, even after an external IP is blocked, it will still be detected (see 2.), in this case a message like `IP to ban: 10.128.0.2  is already in disallowed list.` is printed.
5. **Prometheus metric.** Additionally, it serves Prometheus at http://localhost:8080/metrics with default Golang stats and a counter for the number of attempted connections.

## Why

This is a personal project to learn about [Go's packet decoding library](https://pkg.go.dev/github.com/google/gopacket). This program has no real use-case since the objective of port scan blocking can be done directly with iptables rules, and in any case there are more efficient ways to drop traffic than using iptables.

## Requirements

- There are no special dependencies for this project. The `libpcap-dev` package *may be* required for some Linux distributions.
- For tooling, an [installation script](requirements/debian-ubuntu.sh) is provided. You can download or copy and then run with `bash requirements/debian-ubuntu.sh` (needs access to the internet). This will install in Debian or Ubuntu all you need to build and run this project (except `golangci-lint`): `go`, `git`, `make` and Docker. Note: this script will also set iptables to work in legacy mode in Debian instead of `nftables`.
- Scanbloker has been checked to be compatible (built and ran) with Debian 10.11 and Ubuntu 20.04 LTS (both with amd64 architecture). It should probably work with any or most linux servers with `libpcap` installed. It also runs on Mac with `libpcap` installed (not the 4. iptables part).
- Linux requires root access to work with pcap and accessing network devices, so you need to run it with root privileges (`sudo`).

## Installation

You can 1. build the code or 2. pull an existing Docker image. (Note: the current Docker image has no access to iptables)

1. To build the code, clone this repo `git clone https://github.com/fduran/scanblocker.git && cd scanblocker`. Then you have several options:

  - Build the Golang source code to produce a scanblocker binary.
  To do this, build as any other Go project ('go build' with args), or since a Makefile is provided, run `make build`
  - Build a [distroless](https://github.com/GoogleContainerTools/distroless) Docker image with just the binary, with a 'docker build' command or running `sudo make image`.
  - Modify the existing [ci](.github/workflows/docker-image.yml) GitHub Action so that it builds and pushes the Docker image to a repository of your choice (currently it uses my Docker Hub `fduran` repository).

2. Get the [scanblocker image from Docker Hub](https://hub.docker.com/r/fduran/scanblocker) that this GitHub repository builds automatically, with `docker pull fduran/scanblocker`.

(Note: a proper installation would also involve a systemd service file).

## Usage

There is at the moment only one configuration parameter for the name of the network interface to listen to. This can be set with an environment variable `$SB_DEVICE` or passing a `-i` flag to the command line. The default device is "eth0". Note that if using `sudo`, for the superuser to get your environment you have to use `-E`.

1. Run examples using the binary:

```
# help (if $SB_DEVICE is set, it will show as default)
./scanblocker -h
Usage of ./scanblocker:
  -i string
        interface to listen to, overrides $SB_DEVICE (default "eth0")

# default eth0
sudo ./scanblocker

# argument flag
sudo ./scanblocker -i ens4

# exporting an environment variable
export SB_DEVICE=en0
sudo -E ./scanblocker

# sourcing a possible env var file
. config/.env
sudo -E ./scanblocker
```

Runs until killed (Ctrl-C).

2. Run examples using Docker.

To listen to the host interfaces we need to run docker with the `--net=host` option.  


```
# default eth0 iface
sudo docker run --rm --net=host scanblocker

# passing iface env var
sudo docker run --rm --net=host --env SB_DEVICE=ens4 scanblocker

# we can also pass the device name as a flag in Docker using the internal path to the binary
sudo docker run --rm --net=host scanblocker /app/scanblocker -i ens4
```

## Testing

- Golang unit testing: there's one case for the queue data structure logic.
- [e2e testing](e2e-testing.md)
- Load test: I did a very quick stress test (see below).


### Load test sanity check

We want to run connection attempts repeteadly for a long enough period of timeto check for possible memory leaks or high CPU usage.

Bash has $RANDOM with a range: 0 - 32767 , good for port number testing.

From one server: `while true; do curl $TARGETIP:$RANDOM ; done`

On a GCP e2-small (0.5 CPUs, 2 GB RAM) host target: 
- For the binary: CPU 2% ,  RAM 1.1% or 22 MB
- For the Docker container, same memory but CPU goes up to 4%, there seems to be some network overhead.

I also tried scanning from two servers at the same time but there was no difference in the CPU, RAM usage.  

This is a ridiculous small test sample but it gives a very basic assurance.

## Limitations

(See also the Requirements section)

- The current Docker image has no access to iptables. We would need to install it so current code could find it at the current path and run docker with `--cap-add=NET_ADMIN` enabled.
- IPv4 only.
- Prometheus uses port :8080, so if there's already a service binded to that port then scanblocker wont' start. Prometheus port should be a runtime parameter.
- Limited to 60s and 4 attempts to detect a scan.
- There's no iptables management (like saving rules or logging the rules this code adds).
- Code doesn't take into account the possibility of having multiple IPs per iface device in order to allow-list the local IPs.

## Security Considerations

- The program unfortunately runs as root. In a real deployment we could run it as a user with only the capabilities needed to listen to network interfaces (NET_ADMIN and NET_RAW probably).

- We have to be careful with painting ourself in a corner and "DoS" ourselves by using incorrectly a program than drops all connections from an IP via iptables. For example, we could inadvertedly block legit traffic to a host service. To mitigate this I'm allow-listing the IPs of the host and also testing for repeat connections to the same port.

- DoS. Unperformant code could make things worse if the server is under heavy traffic load (due to a DDoS for example). To mitigate this, I'm filtering only the packets that I need and I'm only reading the necessary number of bytes. Also for scsn detection I'm only keeping track of the last three connections per IP in a queue (timestamp and port, only 10 bytes per connection, maximum 30 per source IP), so the memory usage is minimal. Also I did a very basic stress test to check for CPU usage and possible memory leak.

- Malformed packets. Detecting possibilities like XMAS or NULL scans are out of scope. These maliciously crafted packets could potentially break this program.

- Consider blocking or alerting outgoing scan attempts from the host; this is frequently the sign of a compromised server.
