---
layout: post
title:  "How to Bypass a Corporate VPN with an Android Phone"
date:   2023-09-26
categories: vpn socks proxy termux android
permalink: /how-to-bypass-corporate-vpn
---

Organizations and businesses of all sizes use virtual private networks (VPNs) to provide secure access to their intranet for people working remotely without the risk of exposing their data to the internet. 

Using a VPN client (e.g. Pulse Secure, Cisco AnyConnect, etc.) a device's internet connection is routed through a private server rather than the user's regular internet service provider (ISP), encrypting all the data that is transferred. The ISP will see that the user is sending and receiving data packets, but it won't be able to view their content.

On Linux, the VPN client creates a [tap/tun interface](https://www.kernel.org/doc/Documentation/networking/tuntap.txt), a virtual network interface, where all Ethernet frames or IP packets are routed (to be encrypted) before they are sent to the VPN server through a physical network interface.

This way, a company protect itself against data theft but forces its employees into having no or limited access to the resources commonly available on the internet due to the VPN filtering their traffic. Sites commonly blocked are social networks, music streaming services, any DNS name vaguely sounding suspicious, etc.

Unfortunately, while programming at work behind a VPN sometimes I'm in desperate need of listening to music on YouTube: it helps me to focus on work isolating my mind from surrounding sources of distraction.

An easy mechanism I found out to overcome this limitation is to forward the TCP connection from the browser to my Android phone, which is happily connected to the internet with no restrictions.

In this configuration, the phone acts as a local SOCKS proxy server thanks to OpenSSH which supports [dynamic port forwarding](https://en.wikipedia.org/wiki/Port_forwarding#Dynamic_port_forwarding) (**DPF**). The browser just needs to be instructed on how to use such a proxy to access the internet.

## SOCKS proxy setup

> Disclaimer: please, consider that this could represent a violation of the security policies of your company. Make sure you are not breaking any rules while at work.

First things first, naturally for this mechanism to work the pc and the phone must be connected to the same network (e.g. your home router). 

Then, grab your Android phone and install Termux, a terminal emulator with a minimal GNU/Linux environment, on your phone from [F-Droid](https://wiki.termux.com/wiki/Installing_from_F-Droid).

> Note: do not install it from Google Play since that version is outdated.

Once installed, open the Termux app and install OpenSSH:
```bash
pkg update
pkg upgrade
pkg install root-repo openssh
```

Set a password for the current user by running `passwd` taking note of the password. 

> Note: alternatively, you can use [public key authentication](https://wiki.termux.com/wiki/Remote_Access#Setting_up_public_key_authentication).

Then, simply start the SSH server with `sshd`.

> Note: default SSH port in Termux is `8022`. 

Now switch to your pc.
To open an SSH tunnel with DPF, run the following from your pc:
```bash
$ ssh -D 1337 -q -C -N -f root@phone-ip -p 8022 -o StrictHostKeyChecking=no
# -D 1337, opens a SOCKS proxy on local port 1080
# -C, compresses data in the tunnel, saves bandwidth
# -q, quiet mode, don’t output anything locally
# -N, disallows execution of remote commands, useful for just forwarding ports
# -f, forks the process to background
# -o StrictHostKeyChecking=no, ignores the host key

# type the password set before
root@phone-ip's password: 
```

Configure your browser to use the SOCKS proxy. For example, you can configure the proxy access to the internet in Firefox from the Setting menu as follows:

- select "Manual proxy configuration"
    - enter `localhost` as "SOCKS Host" and the port used above (e.g. `1337`) as "Port"
- tick the option "Proxy DNS when using SOCKS v5"

If you need to use this configuration often, you'd better off creating a specific browser profile in the Firefox Settings so that it's not necessary to constantly switch between different network configurations. A new profile can be created by passing the `-P` flag to the `firefox` command, and launching the Profile Manager:

```bash
$ firefox -P
```

Now, you should be able to freely surf the web.

Finally, when done you can stop the SOCKS proxy by running on Termux:
```bash
pkill sshd
```

## TL;DR

You can use your Android phone as a SOCKS proxy to bypass a VPN filtering your network traffic on your laptop. Once both devices are properly configured, you can open Firefox with a specific profile set to use the SOCKS proxy to access the internet with a single script like the following:
```bash
#!/bin/sh

# To pass the password as an argument to ssh below, install `sshpass`:
# $ sudo apt-get install sshpass
# Please, remember that providing a password on the command line as 
# plain text implies the risk of the password being captured in the user's shell history.
# So, please consider other options to automate the ssh login process.

set -e
 
if [ "$#" -ne 3 ]
then
  echo "USAGE: <script> IP PWD PROFILE" && exit 1
fi

IP=$1
PWD=$2
PROFILE=$3

nohup sshpass -p $PWD ssh -D 1337 -q -C -N -f root@$IP -p 8022 -o StrictHostKeyChecking=no

firefox -P $PROFILE >/dev/null 2>&1 &
```

---

References:
- [Universal TUN/TAP device driver](https://www.kernel.org/doc/Documentation/networking/tuntap.txt)
- [Create a SOCKS proxy on a Linux server with SSH to bypass content filters](https://ma.ttias.be/socks-proxy-linux-ssh-bypass-content-filters/)
- [Termux Wiki. Remote access: Using the SSH server](https://wiki.termux.com/wiki/Remote_Access#Using_the_SSH_server)
