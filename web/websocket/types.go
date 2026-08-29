package websocket

import (
	arkws "goark.dev/arkarta/websocket"
	servletws "goark.dev/arkarta/websocket/servlet"
)

// Endpoint 处理一个 WebSocket 会话的生命周期与消息。
type Endpoint = arkws.Endpoint

// EndpointFunc 将函数组适配为 Endpoint。
type EndpointFunc = arkws.EndpointFunc

// Session 表示一个 WebSocket 会话。
type Session = arkws.Session

// Handshake 表示一次成功的 WebSocket 握手。
type Handshake = arkws.Handshake

// HandshakeOption 定制 WebSocket 握手协商。
type HandshakeOption = arkws.HandshakeOption

// FrameConnectionOption 定制 WebSocket 帧连接。
type FrameConnectionOption = servletws.FrameConnectionOption
