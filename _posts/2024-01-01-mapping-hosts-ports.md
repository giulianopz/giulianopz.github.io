---
layout: post
title:  "What you can do instead of editing /etc/hosts"
date:   2024-01-01
categories: mapping hosts ports localhost
permalink: /what-you-can-do-instead-of-editing-etc-hosts
comments_id: 3
---

When you develop a web application or work with processes exchanging data over a network interface on your local machine, you often stumble upon the issue of: 
- resolving [hostnames and domain names](https://superuser.com/questions/59093/difference-between-host-name-and-domain-name) to localhost
- and optionally, mapping a source port (usually 80, the default for HTTP) to the destination port (your server is listening to).

For example, imagine you have a small Kubernetes cluster (e.g. [k3d](https://k3d.io)) on your machine which needs to pull images from a local image registry. On the other side, you want to push images to that registry too. In both cases, the Docker daemon running in your machine must be able to access the registry by its hostname (let's say `registry.io`).

> Note: you could just reference the registry as `localhost:port` and skip all of the following rigmarole, but sometimes this is not what you want for a number of reasons.

## Quick and Dirty Solution

Usually you would just add an entry to `/etc/hosts` (cf. [hosts(5)](https://linux.die.net/man/5/hosts)) mapping the name of the registry to `127.0.0.1` (i.e. the loopback interface address): 
```bash
$ head -n3 /etc/hosts
127.0.0.1	localhost
127.0.1.1	yourmachinehostname.homenet.isp.com
127.0.0.1	registry.io
```

This is the first source checked by an application that needs to perform name resolution via C library routines such as [getaddrinfo](https://linux.die.net/man/3/getaddrinfo). The next step should be sending DNS queries to the servers listed in the `/etc/resolv.conf` file
(cf. [resolv.conf(5)](https://linux.die.net/man/5/resolv.conf)).

> Note: actually, the order these files are read depends on your Name Service Switch (NSS) configuration file, which is `/etc/nsswitch.conf` (cf. [nsswitch.conf(5)](https://linux.die.net/man/5/nsswitch.conf))

> Note: some programming languages such as [Go](https://pkg.go.dev/net#hdr-Name_Resolution) can use a custom resolver with their own logic as well. 

If you also want to redirect the traffic to a specific port, you have to use [`iptables`](https://www.frozentux.net/iptables-tutorial/iptables-tutorial.html) (or one of its front-ends: firewalld, ufw, etc.):
```bash
# say you want to redirect port 80 to 5000,
# you can create a filter rule as follows
$ iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 5000
```

Otherwise, you can configure the **Apache HTTP Server** (with the [mod_rewrite](https://httpd.apache.org/docs/2.4/mod/mod_rewrite.html) module) or **nginx** (with the [proxy_pass](https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/) directive) to act as a reverse proxy routing incoming requests to the right destination port depending on the server name.

For example, with nginx, if you put the snippet below in `/etc/nginx/conf.d/registry.conf`:
```nginx
server {
    listen 80;

    server_name registry.io;

    location / {
        proxy_pass http://127.0.0.1:5000/;
    }
}
```

You will be able to redirect `http://registry.io` (i.e. port 80 is implied) to `http://127.0.0.1:5000`:
```bash
$ apt install nginx
$ systemctl start nginx
$ curl http://registry.io/v2/_catalog
{"repositories":["debian","alpine"]}
```

Using a reverse proxy has the clear advantage of limiting the port redirection to a specific hostname: this is possible only at the application level, although `iptables` has a `--destination` which you are explicitly discouraged to use for service names (cf. [iptables(8)](https://linux.die.net/man/8/iptables)):

> Hostnames will be resolved once only, before the rule is submitted to the kernel. Please note that specifying any name to be resolved with a remote query such as DNS is a really bad idea.

The problem with this approach is that it tends to clutter your hosts file and it's difficult to maintain in the long run: accidentally forget that you mapped a hostname to localhost in that file and you will be in trouble (if this is not what you want).

I personally reserve the hosts file for blocking shit on the web (ads, malware, etc.) with lists such as the one curated by [Steven Black](https://github.com/StevenBlack/hosts).

> Note: Some people prefer to directly alter the `iptables` filter rules instead of the hosts file, but I can't see any advantages doing that (maybe because I'm not an `iptables` expert).

## Some Better Small Hacks

So, what can you do instead???

### curl

If all you need is just to make a simple HTTP call for a one-shot test, you can instruct [curl](https://everything.curl.dev/usingcurl/connections/name) to resolve a hostname to specific IP address:
```bash
$ curl --resolve registry.io:5000:127.0.0.1 http://registry.io:5000/v2/_catalog
{"repositories":["debian","alpine"]}
```

In fact, the manual (cf. [curl(1)](https://linux.die.net/man/1/curl)) for the flag says:
```
--resolve <[+]host:port:addr[,addr]...>
    Provide  a  custom  address  for a specific host and port pair. Using this, you can make the curl requests(s) use a specified address and prevent the otherwise normally resolved address to be used. Consider it a sort of /etc/hosts alternative provided on the command line. The port number should be the number used for the specific protocol the host will be used for. It means you need several entries if you want to provide address for the same host but different ports.
```

It even allows you to specify both a replacement name and a port number when a specific name and port number is used to connect:
```bash
curl --connect-to registry.io:80:127.0.0.1:5000 http://registry.io/v2/_catalog
{"repositories":["debian","alpine"]}
```

But if you need to test the behaviour of your system(s) in an end-to-end fashion, you will need something more.

### dnsmasq

One option is to use [dnsmasq](https://www.stevenrombauts.be/2018/01/use-dnsmasq-instead-of-etc-hosts/), a lightweight DNS forwarder and DHCP server you can run locally. It requires a bit of configuration though:
```bash
$ apt install dnsmasq
# add 'nameserver 127.0.0.1' at the first line of the following file
$ vi /etc/resolv.conf
# add 'address=/registry.io/127.0.0.1' to a new file in the config dir of dnsmasq
$ vi /etc/dnsmasq.d/dev.conf
$ systemctl restart dnsmasq.service
$ dig +short registry.io @localhost
127.0.0.1
$ curl http://registry.io:5000/v2/_catalog
{"repositories":["debian","alpine"]}
```

### lvh.me

Otherwise, to skip any tedious configurations, you can use a free service named `lvh.me` that simply resolves itself (along with any subdomains you can prefix) to localhost, e.g.:
```bash
$ dig +short registry.lvh.me
127.0.0.1
$ curl http://registry.lvh.me:5000/v2/_catalog
{"repositories":["debian","alpine"]}
```

It's quite popular but it has already [disappeared](https://news.ycombinator.com/item?id=27423225) in the past and nothing guaratees it will still be available in the near future. Interstingly, its author ([levicook](https://github.com/levicook)) does not even mention its existence except for few tweets on its account and a [gist](https://gist.github.com/levicook/563675) he shared 13 years ago... at least, as far as I could find out about it googling around.

> Note: The same functionality is provided by another free service named [localtest.me](http://readme.localtest.me/). 

### nss-myhostname

Luckily, you don't need any external web service if you can install on your Linux distribution the [nss-myhostname](https://man7.org/linux/man-pages/man8/nss-myhostname.8.html) package: a NSS plug-in module that can resolve any subdomains of `localhost` to `127.0.0.1` automatically, so that you can simply refer to a server running locally by adding `.localhost` as a suffix along with its port, e.g.:
```bash
$ apt install nss-myhostname
$ curl http://registry.localhost:5000/v2/_catalog
{"repositories":["debian","alpine"]}
```

In this way everything works with zero configuration on your local environment, except for port redirection (for wich you still need some `iptables` sorcery or a reverse proxy).

## Low-Level Aside

If you would rather redirect network packets at a lower level with your bare hands, you have (at least) two possibilities:

- using [libnetfilter_queue](https://netfilter.org/projects/libnetfilter_queue/) API to hook your user-space program to a numbered netfilter queue (NFQUEUE) where packets are en-queued waiting for a verdict to progress further in the rules chain 

- using [eBPF](https://who.ldelossa.is/posts/ebpf-networking-technique-packet-redirection/) or [XDP](https://www.datadoghq.com/blog/xdp-intro/) to install packet processing programs directly into the kernel itself.

Both for sure come with all the intricacies you have to master when programming below the [Layer 7](https://en.wikipedia.org/wiki/OSI_model#Layer_7:_Application_layer).

I hope to offer some examples in a next blog post.

## TL;DR

Editing `/etc/hosts` to test names of services running locally is a bad idea: you'd better off using it to block unwanted or malicious content on the web. The `nss-myhostname` NSS plug-in module helps you resolving local hostnames (ending with `.localhost`) with zero configuration.

---

References:
- [hosts(5)](https://linux.die.net/man/5/hosts)
- [Apache Module mod_rewrite](https://httpd.apache.org/docs/2.4/mod/mod_rewrite.html)
- [NGINX Reverse Proxy](https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/)
- [Iptables Tutorial](https://www.frozentux.net/iptables-tutorial/iptables-tutorial.html)
- [Name resolve tricks (curl)](https://everything.curl.dev/usingcurl/connections/name)
- [Use dnsmasq instead of /etc/hosts](https://www.stevenrombauts.be/2018/01/use-dnsmasq-instead-of-etc-hosts/)
- [nss-myhostname(8)](https://man7.org/linux/man-pages/man8/nss-myhostname.8.html)
- [Using NFQUEUE and libnetfilter_queue](https://home.regit.org/netfilter-en/using-nfqueue-and-libnetfilter_queue/)
- [XDP Programming Hands-On Tutorial](https://github.com/xdp-project/xdp-tutorial)
- [xdp-redirect demo](https://github.com/zhao-kun/xdp-redirect)
- [eBPF Networking Techniques - Packet Redirection](https://who.ldelossa.is/posts/ebpf-networking-technique-packet-redirection/)
