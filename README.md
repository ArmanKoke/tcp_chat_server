# Chat Server

Server for broadcasting to connected clients and messaging between clients.

## Usage
### Server commands:

```CLIENTS``` - shows list of active clients
- Response:
```
Current active users: @John, @Patrick, @Backer
```
-----
```ALL msg``` - broadcasts **msg** across all active clients
- Example:
```
ALL Welcome to my server!
```

```MSG @recipient msg``` - sends **msg** to tagged **recipient**
- Example:
```
MSG @Patrick You are Banned!
```

### Client commands:

```MSG @recipient msg``` - sends **msg** to tagged **recipient**
- Example:
```
MSG @John Hey sup!
```
## Note!
If you don't see ```Type command:``` just press *Enter* key

Server is used with following [Client](https://github.com/ArmanKoke/tcp_chat_client)

Or make new one by yourself because it has certain format for commands. 
