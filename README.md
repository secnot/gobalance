# Bitcoin balance micro service (Alpha)

Gobalance is a bitcoin address balance micro service written in Go, designed from the ground up
to be deployed in a resizable cluster of inexpensive vps. Each node just keeps an up to date utxo 
table that is used to obtain the balance for the addresses.

It IS NOT a full node it just uses bitcoind rpc to obtain the latest blocks to update the utxo
table.

The objective is to be able to deploy gobalance in small servers with around 20GB of free storage 
and less than 2GB of RAM, in just a few minutes using docker.


Currently in alpha testing


## Config Options

Go balance can be configured in two ways, using the configuration file **$HOME/.gobalance/conf.toml** 
or through environment vars. In the second case any env var will shadow the same option in the file.
[See More](config/README.md)
