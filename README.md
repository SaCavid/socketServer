# SocketServer

Socket server for Machines and Users

## Description

This is socket server project where admin users can control machine clients. Admin users join server via WebSocket while
machine clients use TCP protocol Implemented HUB and Message broker allow users send commands to machines

## Installation

to start project in Linux system. Golang 1.65+ must be installed . After installation command below will start app.

```
mkdir project_folder
cd project_folder
git clone https://gitlab.com/sadiqov.cavid/socketserver.git
git branch -M main
 
go build -o main.go
./main 
 
```

## Websocket server

Websocket will start in default http://localhost:8080 address To change host and port .env file must be changed

## Test and Deploy

For testing go to url http://localhost:8080/. Connect to WebSocket and send command to connected workers.

![terminal example](docs/terminal.png)

***

# README

Read file used as manual for this app. We can add any question and save here.

## App structure

For better understanding of inapp functionality check https://draw.io files in docs folder

![terminal example](docs/drawio.png)

## Usage

## Recommended

    - Add Authorization to websocket connection
    - Add Ping/Pong protocol to App and machile clients.

## Project status

Project in testing stage
