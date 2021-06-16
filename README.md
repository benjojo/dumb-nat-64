dumb-nat-64
===


People talk of many ways of doing NAT64, I have a stupid way of doing it for low volume stuff, like trying to get to github from v6 only containers/VMs.

This NAT64+DNS64 trick only works for TCP services only, but also given that most corp networks look like that anyway. this should be fine.

The core of this service is the dumb-nat-64 binary, that neets a iptables redirect inside.

```
ip6tables -A PREROUTING -d 3000::/96 -i eth0 -p tcp -j REDIRECT --to-ports 1337
```

and root (I think??). After that you can just install it as a systemd service.

```
[Unit]
Description=NAT64 server
After=network.target auditd.service

[Service]
ExecStart=/usr/bin/dumb-nat-64
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

In order to have DNS translated over, consider also running a coredns server with the following config:

```
. {
    dns64 3000::/96
    forward . 8.8.8.8
    log
    errors
}
```

---

Routing (and firewalling of the coredns process) is left up to the reader. the 3000:: prefix is flexible, and you can get away with changing it as long as the address is at the end of the ipv6 target prefix.

In my setup I have 3000::/32 routed into a LXC container, that has coredns listening on 3000::53, and the same ip6table rule above. bird OSPF ensures that packets get "sucked into" the container correctly anywhere in the network.