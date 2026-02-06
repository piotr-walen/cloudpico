module cloudpico-server

go 1.25.6

require (
	cloudpico-shared v0.0.0
	github.com/eclipse/paho.mqtt.golang v1.5.1
	github.com/lmittmann/tint v1.1.2
	github.com/mattn/go-sqlite3 v1.14.33
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
)

replace cloudpico-shared => ../shared
