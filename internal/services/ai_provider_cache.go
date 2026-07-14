package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/yottaapp/yotta/internal/securestore"
	"github.com/yottaapp/yotta/internal/services/llm"
)

// SettingsGetter 取当前 Settings(live)。缓存取用时调它,使 settings 变更自然反映到指纹。
type SettingsGetter func() *Settings

type aiCacheEntry struct {
	provider llm.Provider
	fp       string
}

// AIProviderCache 按 connectionID 缓存 llm.Provider。取用时按连接指纹自愈:
// settings 改了 endpoint/key → 指纹不符 → 重建(无事件监听)。改 Label 不影响指纹。
type AIProviderCache struct {
	get     SettingsGetter
	secrets *AISecrets
	mu      sync.Mutex
	cache   map[string]aiCacheEntry
}

func NewAIProviderCache(get SettingsGetter, secrets ...*AISecrets) *AIProviderCache {
	cache := &AIProviderCache{get: get, cache: map[string]aiCacheEntry{}}
	if len(secrets) > 0 {
		cache.secrets = secrets[0]
	}
	return cache
}

// Provider 解析 connectionID(空 → ai.default)并返该连接的 Provider(缓存)。
func (c *AIProviderCache) Provider(connectionID string) (llm.Provider, error) {
	conn := c.resolve(c.get(), connectionID)
	if conn == nil {
		return nil, &llm.CodedError{Kind: llm.KindNotFound, Err: fmt.Errorf("ai connection %q not found", connectionID)}
	}
	apiKey := conn.APIKey
	if c.secrets != nil {
		stored, err := c.secrets.Get(conn.ID)
		if err == nil {
			apiKey = stored
		} else if !errors.Is(err, securestore.ErrNotFound) && !errors.Is(err, securestore.ErrUnavailable) {
			return nil, fmt.Errorf("read AI credential: %w", err)
		}
	}
	fp := connFingerprint(conn, apiKey)

	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.cache[conn.ID]; ok && e.fp == fp {
		return e.provider, nil
	}
	p, err := llm.New(llm.ConnectionConfig{Protocol: conn.Protocol, BaseURL: conn.BaseURL, APIKey: apiKey})
	if err != nil {
		return nil, err
	}
	if old, ok := c.cache[conn.ID]; ok {
		if cl, ok := old.provider.(interface{ CloseIdleConnections() }); ok {
			cl.CloseIdleConnections()
		}
	}
	c.cache[conn.ID] = aiCacheEntry{provider: p, fp: fp}
	return p, nil
}

func (c *AIProviderCache) resolve(s *Settings, id string) *AIConnection {
	if id == "" {
		return s.DefaultConnection()
	}
	for i := range s.AI.Connections {
		if s.AI.Connections[i].ID == id {
			return &s.AI.Connections[i]
		}
	}
	return nil
}

// connFingerprint 只 hash 影响 llm.New 行为的字段(Protocol/BaseURL/APIKey),
// 排除 Label/ID —— 改名/调序不该触发 Provider 重建。
func connFingerprint(c *AIConnection, apiKey string) string {
	h := sha256.Sum256([]byte(c.Protocol + "\x00" + c.BaseURL + "\x00" + apiKey))
	return hex.EncodeToString(h[:])
}
