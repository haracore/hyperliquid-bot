// Package websocket mirrors hyperliquid-python-sdk/hyperliquid/websocket_manager.py.
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Subscription mirrors the Python Subscription TypedDict union as a dynamic map.
type Subscription map[string]any

// Callback mirrors Python callback signature Callable[[Any], None].
type Callback func(message map[string]any)

type activeSubscription struct {
	callback       Callback
	subscriptionID int
}

// WebsocketManager corresponds to Python:
// hyperliquid.websocket_manager.WebsocketManager
type WebsocketManager struct {
	baseURL string
	wsURL   string

	mu                    sync.Mutex
	conn                  *websocket.Conn
	subscriptionIDCounter int
	wsReady               bool
	queuedSubscriptions   []queuedSubscription
	activeSubscriptions   map[string][]activeSubscription
}

type queuedSubscription struct {
	subscription Subscription
	active       activeSubscription
}

// NewWebsocketManager corresponds to Python:
// hyperliquid.websocket_manager.WebsocketManager.__init__
func NewWebsocketManager(baseURL string) *WebsocketManager {
	wsURL := "ws" + strings.TrimPrefix(baseURL, "http") + "/ws"
	return &WebsocketManager{
		baseURL:             baseURL,
		wsURL:               wsURL,
		activeSubscriptions: map[string][]activeSubscription{},
	}
}

// Start connects and starts read and ping loops.
func (m *WebsocketManager) Start(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, m.wsURL, nil)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.conn = conn
	m.wsReady = true
	queued := append([]queuedSubscription{}, m.queuedSubscriptions...)
	m.queuedSubscriptions = nil
	m.mu.Unlock()

	for _, item := range queued {
		if err := m.subscribeReady(item.subscription, item.active); err != nil {
			return err
		}
	}
	go m.readLoop()
	go m.pingLoop(ctx)
	return nil
}

// Stop corresponds to Python:
// hyperliquid.websocket_manager.WebsocketManager.stop
func (m *WebsocketManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil {
		return nil
	}
	err := m.conn.Close()
	m.conn = nil
	m.wsReady = false
	return err
}

func (m *WebsocketManager) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = m.writeJSON(map[string]any{"method": "ping"})
		}
	}
}

func (m *WebsocketManager) readLoop() {
	for {
		_, data, err := m.conn.ReadMessage()
		if err != nil {
			return
		}
		if string(data) == "Websocket connection established." {
			continue
		}
		var msg map[string]any
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		identifier := WsMsgToIdentifier(msg)
		if identifier == "" || identifier == "pong" {
			continue
		}
		m.mu.Lock()
		subs := append([]activeSubscription{}, m.activeSubscriptions[identifier]...)
		m.mu.Unlock()
		for _, sub := range subs {
			sub.callback(msg)
		}
	}
}

func (m *WebsocketManager) writeJSON(value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil {
		return nil
	}
	return m.conn.WriteJSON(value)
}

// SubscriptionToIdentifier corresponds to Python:
// hyperliquid.websocket_manager.subscription_to_identifier
func SubscriptionToIdentifier(subscription Subscription) string {
	typ, _ := subscription["type"].(string)
	switch typ {
	case "allMids":
		return "allMids"
	case "l2Book", "trades", "bbo", "activeAssetCtx":
		return fmt.Sprintf("%s:%s", typ, lowerString(subscription["coin"]))
	case "userEvents":
		return "userEvents"
	case "userFills", "userFundings", "userNonFundingLedgerUpdates", "webData2":
		return fmt.Sprintf("%s:%s", typ, lowerString(subscription["user"]))
	case "candle":
		return fmt.Sprintf("candle:%s,%s", lowerString(subscription["coin"]), subscription["interval"])
	case "orderUpdates":
		return "orderUpdates"
	case "activeAssetData":
		return fmt.Sprintf("activeAssetData:%s,%s", lowerString(subscription["coin"]), lowerString(subscription["user"]))
	default:
		return ""
	}
}

// WsMsgToIdentifier corresponds to Python:
// hyperliquid.websocket_manager.ws_msg_to_identifier
func WsMsgToIdentifier(wsMsg map[string]any) string {
	channel, _ := wsMsg["channel"].(string)
	switch channel {
	case "pong":
		return "pong"
	case "allMids":
		return "allMids"
	case "l2Book":
		return "l2Book:" + lowerNestedString(wsMsg, "data", "coin")
	case "trades":
		trades, _ := wsMsg["data"].([]any)
		if len(trades) == 0 {
			return ""
		}
		first, _ := trades[0].(map[string]any)
		return "trades:" + lowerString(first["coin"])
	case "user":
		return "userEvents"
	case "userFills":
		return "userFills:" + lowerNestedString(wsMsg, "data", "user")
	case "candle":
		data, _ := wsMsg["data"].(map[string]any)
		return fmt.Sprintf("candle:%s,%s", lowerString(data["s"]), data["i"])
	case "orderUpdates":
		return "orderUpdates"
	case "userFundings":
		return "userFundings:" + lowerNestedString(wsMsg, "data", "user")
	case "userNonFundingLedgerUpdates":
		return "userNonFundingLedgerUpdates:" + lowerNestedString(wsMsg, "data", "user")
	case "webData2":
		return "webData2:" + lowerNestedString(wsMsg, "data", "user")
	case "bbo":
		return "bbo:" + lowerNestedString(wsMsg, "data", "coin")
	case "activeAssetCtx", "activeSpotAssetCtx":
		return "activeAssetCtx:" + lowerNestedString(wsMsg, "data", "coin")
	case "activeAssetData":
		data, _ := wsMsg["data"].(map[string]any)
		return fmt.Sprintf("activeAssetData:%s,%s", lowerString(data["coin"]), lowerString(data["user"]))
	default:
		return ""
	}
}

func lowerNestedString(msg map[string]any, parent string, child string) string {
	data, _ := msg[parent].(map[string]any)
	return lowerString(data[child])
}

// Subscribe corresponds to Python:
// hyperliquid.websocket_manager.WebsocketManager.subscribe
func (m *WebsocketManager) Subscribe(subscription Subscription, callback Callback, subscriptionID ...int) (int, error) {
	m.mu.Lock()
	id := 0
	if len(subscriptionID) > 0 {
		id = subscriptionID[0]
	} else {
		m.subscriptionIDCounter++
		id = m.subscriptionIDCounter
	}
	active := activeSubscription{callback: callback, subscriptionID: id}
	if !m.wsReady {
		m.queuedSubscriptions = append(m.queuedSubscriptions, queuedSubscription{subscription: subscription, active: active})
		m.mu.Unlock()
		return id, nil
	}
	m.mu.Unlock()
	return id, m.subscribeReady(subscription, active)
}

func (m *WebsocketManager) subscribeReady(subscription Subscription, active activeSubscription) error {
	identifier := SubscriptionToIdentifier(subscription)
	m.mu.Lock()
	if (identifier == "userEvents" || identifier == "orderUpdates") && len(m.activeSubscriptions[identifier]) != 0 {
		m.mu.Unlock()
		return fmt.Errorf("cannot subscribe to %s multiple times", identifier)
	}
	m.activeSubscriptions[identifier] = append(m.activeSubscriptions[identifier], active)
	m.mu.Unlock()
	return m.writeJSON(map[string]any{"method": "subscribe", "subscription": subscription})
}

// Unsubscribe corresponds to Python:
// hyperliquid.websocket_manager.WebsocketManager.unsubscribe
func (m *WebsocketManager) Unsubscribe(subscription Subscription, subscriptionID int) (bool, error) {
	m.mu.Lock()
	if !m.wsReady {
		m.mu.Unlock()
		return false, fmt.Errorf("can't unsubscribe before websocket connected")
	}
	identifier := SubscriptionToIdentifier(subscription)
	active := m.activeSubscriptions[identifier]
	next := make([]activeSubscription, 0, len(active))
	removed := false
	for _, sub := range active {
		if sub.subscriptionID == subscriptionID {
			removed = true
			continue
		}
		next = append(next, sub)
	}
	m.activeSubscriptions[identifier] = next
	shouldUnsubscribe := len(next) == 0
	m.mu.Unlock()
	if shouldUnsubscribe {
		return removed, m.writeJSON(map[string]any{"method": "unsubscribe", "subscription": subscription})
	}
	return removed, nil
}

// WSURL returns the websocket URL. It exists to make tests and port reviews simple.
func (m *WebsocketManager) WSURL() string {
	u, err := url.Parse(m.wsURL)
	if err != nil {
		return m.wsURL
	}
	return u.String()
}

func lowerString(v any) string {
	s, _ := v.(string)
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			b := []byte(s)
			for j := i; j < len(b); j++ {
				if b[j] >= 'A' && b[j] <= 'Z' {
					b[j] += 'a' - 'A'
				}
			}
			return string(b)
		}
	}
	return s
}
