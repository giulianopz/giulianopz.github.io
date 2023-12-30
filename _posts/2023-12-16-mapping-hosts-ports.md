---
layout: post
title:  "Name Resolution and Network Packets Redirection on Localhost"
date:   2023-12-16
categories: mapping hosts ports localhost
permalink: /mapping-hosts-and-ports-to-localhost
comments_id: 3
---

When you develop a web server on a local environment you often stumble upon the issue of resolving [hostnames and domain names](https://superuser.com/questions/59093/difference-between-host-name-and-domain-name) to localhost. 

For example, you may have a small Kubernetes cluster (e.g. [k3d](https://k3d.io)) on your machine which needs to access a local image registry by its hostname: you can't directly reference it via `localhost` since it would be interpreted *locally* to the Docker container running the cluster.

The quick and dirty solution for it is to add an entry to `/etc/hosts` mapping a domain name to `127.0.0.1`. This is the first source usually (it depends on your Name Service Switch ([NSS](https://linux.die.net/man/5/nsswitch.conf)) configuration file, `/etc/nsswitch.conf`) consulted by an application that needs to perform name resolution via C library routines such as [getaddrinfo](https://linux.die.net/man/3/getaddrinfo), before sending DNS queries to the servers listed in the [/etc/resolv.conf](https://linux.die.net/man/5/resolv.conf) file.

> Note: some programming languages as [Go](https://pkg.go.dev/net#hdr-Name_Resolution) can use a custom resolver with their own logic as well. 

But if you want to redirect the traffic to a specific port, you also need to use a reverse proxy such as the Apache HTTP Server ([mod_rewrite](https://httpd.apache.org/docs/2.4/mod/mod_rewrite.html)) or `nginx` ([proxy_pass](https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/)).

To configure nginx, put the snippet below in `/etc/nginx/conf.d/yourdomain.conf`:
```nginx
server {
    listen 80;

    server_name yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:8080/;
    }
}
```

And you will be able to redirect `yourdomain.com` to `127.0.0.1:8080`, provided that the client is proxy-aware (e.g reading from the environment the variables `http_proxy` and/or `https_proxy`).

This approach tends to clutter your hosts file and it's difficult to maintain: it can cause you troubles if you accidentally forget that you are not able to resolve a domain name just because you are mapping it to your localhost.

Some people prefer altering the [iptables](https://www.frozentux.net/iptables-tutorial/iptables-tutorial.html) rules...

If all you need is just to make a simple HTTP call for a one-shot test, you can instruct [curl](https://everything.curl.dev/usingcurl/connections/name) to resolve a specific IP address for a host name:
```bash
curl --resolve example.com:80:127.0.0.1 http://example.com/
```

In fact, the documentation for the flag says:
```
--resolve <[+]host:port:addr[,addr]...>
    Provide  a  custom  address  for a specific host and port pair. Using this, you can make the curl requests(s) use a specified address and prevent the otherwise normally resolved address to be used. Consider it a sort of /etc/hosts alternative provided on the command line. The port number should be the number used for the specific protocol the host will be used for. It means you need several entries if you want to provide address for the same host but different ports.
```

It even allows you to specify a replacement name and a port number when a specific name and port number is used to connect:
```bash
curl --connect-to www.example.com:80:load1.example.com:80 http://www.example.com
```

Otherwise, the [nss-myhostname](https://man7.org/linux/man-pages/man8/nss-myhostname.8.html) package can resolve any subdomains of localhsot to the loopback interface automatically.

The same functionality is offered by a free service named `lvh.me` that simply resolves itself with any subdomains you can prefix it to your localhost. It's quite popular but it has already [disappeared](https://news.ycombinator.com/item?id=27423225) in the past and nothing guaratees it will be available in the near future. Interstingly, its author, [levincook](https://github.com/levicook), does not even mention its existence except for few tweets on its account and a [gist](https://gist.github.com/levicook/563675) he shared 13 years ago...

If you want to redirect network packets at a lower level with your bare hands, you have (at least) two possibilities:
- using [libnetfilter_queue](https://netfilter.org/projects/libnetfilter_queue/) API in your userspace program to process the packets that have been queued by means of a iptables rule redirecting the traffic to the NFQUEUE target
- using [eBPF](https://who.ldelossa.is/posts/ebpf-networking-technique-packet-redirection/) or [XDP](https://www.datadoghq.com/blog/xdp-intro/) to install packet processing programs directly into the kernel itself.

This for sure come with all the intricacies you have to master when programming below the [Layer 7](https://en.wikipedia.org/wiki/OSI_model#Layer_7:_Application_layer). 

---

References:
- [Apache Module mod_rewrite](https://httpd.apache.org/docs/2.4/mod/mod_rewrite.html)
- [NGINX Reverse Proxy](https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/)
- [Name resolve tricks (curl)](https://everything.curl.dev/usingcurl/connections/name)
- [nss-myhostname(8)](https://man7.org/linux/man-pages/man8/nss-myhostname.8.html)
- [Using NFQUEUE and libnetfilter_queue](https://home.regit.org/netfilter-en/using-nfqueue-and-libnetfilter_queue/)
- [XDP Programming Hands-On Tutorial](https://github.com/xdp-project/xdp-tutorial)
- [xdp-redirect demo](https://github.com/zhao-kun/xdp-redirect)
- [eBPF Networking Techniques - Packet Redirection](https://who.ldelossa.is/posts/ebpf-networking-technique-packet-redirection/)
