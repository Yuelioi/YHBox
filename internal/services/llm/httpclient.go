package llm

import "net/http"

// tunedHTTPClient 返回带连接池的 client: 复用 keep-alive 空闲连接、IdleConnTimeout 自动回收。
// 不设 Client.Timeout —— 流式响应会被它掐断; 超时一律走 per-call ctx deadline。
func tunedHTTPClient() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	return &http.Client{Transport: t}
}
