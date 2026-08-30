package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

const (
	// SSE 连接上限：防止恶意客户端大量建立长连接耗尽内存
	maxSSEClientsPerUser = 8
	maxSSEClientsTotal   = 5000
)

type SSEClient struct {
	ID       int
	Messages chan []byte
}

type SSEBroker struct {
	mu      sync.RWMutex
	clients map[int]map[string]*SSEClient // userId -> clientId -> client
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: make(map[int]map[string]*SSEClient),
	}
}

// Subscribe 注册 SSE 连接。超过单用户/全局连接上限时返回错误。
func (b *SSEBroker) Subscribe(userId int, clientId string) (*SSEClient, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.clients) >= maxSSEClientsTotal {
		return nil, errors.New("SSE 连接数已达全局上限")
	}
	if len(b.clients[userId]) >= maxSSEClientsPerUser {
		return nil, errors.New("该用户的 SSE 连接数已达上限")
	}
	client := &SSEClient{ID: userId, Messages: make(chan []byte, 64)}
	if b.clients[userId] == nil {
		b.clients[userId] = make(map[string]*SSEClient)
	}
	b.clients[userId][clientId] = client
	slog.Info("SSE client connected", "userId", userId, "clientId", clientId)
	return client, nil
}

func (b *SSEBroker) Unsubscribe(userId int, clientId string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.clients[userId] != nil {
		delete(b.clients[userId], clientId)
		if len(b.clients[userId]) == 0 {
			delete(b.clients, userId)
		}
	}
	slog.Info("SSE client disconnected", "userId", userId, "clientId", clientId)
}

func (b *SSEBroker) SendToUser(userId int, eventType string, data interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.clients[userId] == nil {
		return
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	msg := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, payload))
	for _, client := range b.clients[userId] {
		select {
		case client.Messages <- msg:
		default:
			// channel full, skip（未读通知计数可作补偿通道）
		}
	}
}
