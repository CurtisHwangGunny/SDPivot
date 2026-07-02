// smartKnora integration is now handled via proper DI container.
// See: internal/container/container.go (line: router.NewSmartKnoraRouter)
// See: internal/router/smartknora.go (SmartKnoraRouter + RegisterRoutes)
//
// The SmartKnoraRouter is injected into WeKnora's RouterParams via dig
// and routes are registered in NewRouter() automatically.
//
// JWT secret is configured via SMARTKNORA_JWT_SECRET env var.
package main
