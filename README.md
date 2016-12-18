# go_rps
## Reverse Proxy Service

So you wanna expose a server that is otherwise publicly inaccessible?

This reverse proxy service will provide a publicly accessible url that can be used to access your hidden server.


## Steps to use (Command Line Interface)
Assuming the Go environment is correctly set up on your machine...

1. Install the go_rps client command line interface
  1. cd rps_cli
  2. go get -t
  3. go install

2. Assume you have a server running locally @ localhost:\<SOME_PORT\> that you want to expose
3. Start the rps client using the CLI to connect to our rps_server (you can also connect to your own, see below)
  1. rps_cli \<SOME_PORT\> \<RPS_SERVER_URL\>
  2. Our server url is: 45.33.109.4:34567
4. The CLI will output "Tunnel opened! Go here: \<PUBLIC_URL\>"
5. Now you can use \<PUBLIC_URL\> to access your server, either through a browser or a TCP connection!

## Steps to use (Go library)
Assuming you have a server running locally @ localhost:\<SOME_PORT\> that you want to expose

1. Create a client, passing it the rps server address (45.33.109.4:34567, or you can host your own below)
```go
serverAddress := &net.TCPAddr{
  IP: net.IPv4(45, 33, 109, 4),
  Port: 34567,
}

client := go_rps.GoRpsClient{
  ServerTCPAddr: serverAddress,
}
```
2. Open a tunnel to your hidden server running on localhost:\<SOME_PORT\> and extract the exposed address.
```go
err = client.OpenTunnel(SOME_PORT)
if err != nil {
  log.Printf("Unable to open tunnel.\n")
  return nil
}

exposedTCPAddr := *serverAddress
exposedTCPAddr.Port = client.ExposedPort
log.Printf("Go here: %s\n", exposedTCPAddr.String())
```
3. The exposed address now accepts TCP connections and will route data to and from the hidden server!

## Run your own server

1. Compile a go binary for the OS that will be running your server
  1. e.g. for linux: env GOOS=linux go build -o main.linux main.go
  2. run ./main.linux on your host
  3. The rps server will run on port 34567

## How it works

Magic
