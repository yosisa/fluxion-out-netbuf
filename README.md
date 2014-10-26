# fluxion-out-netbuf
[![wercker status](https://app.wercker.com/status/d30fa7629b0108d95dc3996d6be56cf5/s/master "wercker status")](https://app.wercker.com/project/bykey/d30fa7629b0108d95dc3996d6be56cf5)

fluxion-out-netbuf is an output plugin for [Fluxion](https://github.com/yosisa/fluxion), which provides network buffer for each client. The plugin listen specified tcp port and waiting for incoming connections. Once a connection has come, the plugin creates dedicated buffer for that connection's source IP address and closes the connection. Since this time, incoming fluxion's events are stored in that buffer in order to send to subsequent connection from same IP address. This is especially useful for monitoring error logs by checking tcp response with periodic way.

## Installation
```
go get github.com/yosisa/fluxion-out-netbuf
```
