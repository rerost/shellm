# shellm
Run any command by ChatGPT

## Example
```
> Create VPN
[ChatGPT] Which VPN Type? (L2TLP or ....)
1. L2TLP
2. Cisco ...

> 1
[ChatGPT] Creating GPT
$ sudo apt-get update
$ sudo apt-get install -y strongswan-ikev1
$ echo "..." > /etc/...
$ sudo systemctl enable pppd ...
$ sudo systemctl restart strongswan-starter.service  && sudo ipsec reload && sudo systemctl restart xl2tpd

[ChatGPT] Checking...
$ sudo ipsec up L2TP-PSK
$ ...
[ChatGPT] Retrying...
$ echo "..." > /etc/...
$ sudo systemctl restart strongswan-starter.service  && sudo ipsec reload && sudo systemctl restart xl2tpd
[ChatGPT] Checking...
$ sudo ipsec up L2TP-PSK
$ ...

> Yes
```