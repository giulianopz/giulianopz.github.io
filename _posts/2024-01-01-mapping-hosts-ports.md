---
layout: post
title:  "Mapping Hosts and Ports to Localhost"
date:   2024-01-01
categories: mapping hosts ports localhost
permalink: /mapping-hosts-and-ports-to-localhost
comments_id: 3
---

When you develop a web application or work with processes exchanging data over a network interface on your local machine, you often stumble upon the issues of: 
- resolving [hostnames and domain names](https://superuser.com/questions/59093/difference-between-host-name-and-domain-name) to localhost
- and optionally, mapping a source port (usually 80, the default for HTTP) to the destination port (your server is listening to).

For example, you may have a small Kubernetes cluster (e.g. [k3d](https://k3d.io)) on your machine which needs to pull images from a local image registry by its hostname (let's say `registry.io`). Vice versa, you want to push images to that registry by its name from your local machine, which does not know how to resolve the registry name. In both cases, the docker daemon running in your machine must be able to access the registry.

> Note: you could just use `localhost:port` and skip all of the following rigmarole, but sometimes it's not what you want.

## Quick and Dirty Solution

The quick and dirty solution for it is to add an entry to [`/etc/hosts`](https://linux.die.net/man/5/hosts) mapping a domain name to `127.0.0.1`: 
```bash
$ head -n3 /etc/hosts
127.0.0.1	localhost
127.0.1.1	yourmachinehostname.homenet.isp.com
127.0.0.1	registry.io
```

Usually this is the first source (it depends on your Name Service Switch ([NSS](https://linux.die.net/man/5/nsswitch.conf)) configuration file, `/etc/nsswitch.conf`) checked by an application that needs to perform name resolution via C library routines such as [getaddrinfo](https://linux.die.net/man/3/getaddrinfo). The next step should be sending DNS queries to the servers listed in the [/etc/resolv.conf](https://linux.die.net/man/5/resolv.conf) file.

> Note: some programming languages as [Go](https://pkg.go.dev/net#hdr-Name_Resolution) can use a custom resolver with their own logic as well. 

This works fine but if you want to redirect the traffic to a specific port, you will also need to setup a reverse proxy: you can configure the Apache HTTP Server (with the [mod_rewrite](https://httpd.apache.org/docs/2.4/mod/mod_rewrite.html) module) or `nginx` (with the [proxy_pass](https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/) directive) to proxy the incoming requests to the right destination port.

To configure nginx, you could put the snippet below in `/etc/nginx/conf.d/registry.conf`:
```nginx
server {
    listen 80;

    server_name registry.io;

    location / {
        proxy_pass http://127.0.0.1:5000/;
    }
}
```

And you would be able to redirect `registry.io` to `127.0.0.1:5000` (provided that you set up a [local docker registry](https://www.natarajmb.com/2021/10/kubernetes-local-development-k3d-docker/)):
```bash
$ curl http://registry.io/v2/_catalog
{"repositories":["debian","alpine"]}
```

This approach tends to clutter your hosts file and it's difficult to maintain in the long run: accidentally forget that you mapped a hostname to localhost in that file and you will be in trouble (if this is not what you want).

> Note: Some people prefer to directly alter the [iptables](https://www.frozentux.net/iptables-tutorial/iptables-tutorial.html) rules instead of the hosts file, but I can't see any advantages doing that (maybe because I'm not an iptables expert).

## A Better Small Hack

If all you need is just to make a simple HTTP call for a one-shot test, you can instruct [curl](https://everything.curl.dev/usingcurl/connections/name) to resolve a hostname to specific IP address:
```bash
$ curl --resolve registry.io:5000:127.0.0.1 http://registry.io:5000/v2/_catalog
{"repositories":["debian","alpine"]}
```

In fact, the documentation for the flag says:
```
--resolve <[+]host:port:addr[,addr]...>
    Provide  a  custom  address  for a specific host and port pair. Using this, you can make the curl requests(s) use a specified address and prevent the otherwise normally resolved address to be used. Consider it a sort of /etc/hosts alternative provided on the command line. The port number should be the number used for the specific protocol the host will be used for. It means you need several entries if you want to provide address for the same host but different ports.
```

It even allows you to specify both a replacement name and a port number when a specific name and port number is used to connect:
```bash
curl --connect-to registry.io:80:127.0.0.1:5000 http://registry.io/v2/_catalog
{"repositories":["debian","alpine"]}
```

But if you need to test the behaviour of your system(s) end-to-end, you will need something more.

There's a free service named `lvh.me` that simply resolves itself with any subdomains you can prefix to your localhost, e.g.:
```bash
$ dig +short registry.lvh.me
127.0.0.1
$ curl http://registry.lvh.me:5000/v2/_catalog
{"repositories":["debian","alpine"]}
```

It's quite popular but it has already [disappeared](https://news.ycombinator.com/item?id=27423225) in the past and nothing guaratees it will be available in the near future. Interstingly, its author ([levincook](https://github.com/levicook)) does not even mention its existence except for few tweets on its account and a [gist](https://gist.github.com/levicook/563675) he shared 13 years ago... at least, as far as I could find out about it googling around.

Under the hood, it redirects any requests to your machine thanks to a small DNS trick. The same functionality is provided by another free service named [localtest.me](http://readme.localtest.me/). 

Luckily, you don't need any external web service if you can install on your Linux distribution the [nss-myhostname](https://man7.org/linux/man-pages/man8/nss-myhostname.8.html) package: it can resolve any subdomains of localhost to the `127.0.0.1` automatically, so that you can simply refer to a server running locally by adding `.localhost` as a suffix along with its port, e.g.:
```bash
$ curl http://registry.localhost:5000/v2/_catalog
{"repositories":["debian","alpine"]}
```

In this way (almost) everything works with zero configuration on your local environment, and so it's even better than using [dnsmasq](https://www.stevenrombauts.be/2018/01/use-dnsmasq-instead-of-etc-hosts/) which requires a bit of configuration to get the same result.

That said, your are still in need of specifying the target port. You will probably need `iptables` (or `ssh`) to setup a port forwarding if that is crucial for you. 

## Low-Level Solution

If you would like to redirect network packets at a lower level with your bare hands, you have (at least) two possibilities:

- using [libnetfilter_queue](https://netfilter.org/projects/libnetfilter_queue/) API to hook your userspace program to a numbered netfilter queue (NFQUEUE) where packets are en-queued waiting for a verdict to progress further in the rules chain 

- using [eBPF](https://who.ldelossa.is/posts/ebpf-networking-technique-packet-redirection/) or [XDP](https://www.datadoghq.com/blog/xdp-intro/) to install packet processing programs directly into the kernel itself.

Both for sure come with all the intricacies you have to master when programming below the [Layer 7](https://en.wikipedia.org/wiki/OSI_model#Layer_7:_Application_layer).

I hope to offer some examples in a next blog post.

---

References:
- [Apache Module mod_rewrite](https://httpd.apache.org/docs/2.4/mod/mod_rewrite.html)
- [NGINX Reverse Proxy](https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/)
- [Iptables Tutorial](https://www.frozentux.net/iptables-tutorial/iptables-tutorial.html)
- [Name resolve tricks (curl)](https://everything.curl.dev/usingcurl/connections/name)
- [nss-myhostname(8)](https://man7.org/linux/man-pages/man8/nss-myhostname.8.html)
- [Use dnsmasq instead of /etc/hosts](https://www.stevenrombauts.be/2018/01/use-dnsmasq-instead-of-etc-hosts/)
- [Using NFQUEUE and libnetfilter_queue](https://home.regit.org/netfilter-en/using-nfqueue-and-libnetfilter_queue/)
- [XDP Programming Hands-On Tutorial](https://github.com/xdp-project/xdp-tutorial)
- [xdp-redirect demo](https://github.com/zhao-kun/xdp-redirect)
- [eBPF Networking Techniques - Packet Redirection](https://who.ldelossa.is/posts/ebpf-networking-technique-packet-redirection/)
