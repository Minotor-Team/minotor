import express from 'express';
import { createServer } from 'http';
import { Server, OPEN } from 'ws';

const port = 6969;
const server = createServer(express);
const wss = new Server({ server })

wss.on('connection', function connection(ws) {
    ws.on('message', function incoming(data) {
        wss.clients.forEach(function each(client) {
            if (client !== ws && client.readyState === OPEN) {
                client.send(data);
            }
        })
    })
})

server.listen(port, function () {
    console.log(`Server is listening on ${port}!`)
})