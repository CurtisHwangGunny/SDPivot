// SDPivot integration is now handled via proper DI container.
// See: internal/container/container.go (line: router.NewSDPivotRouter)
// See: internal/router/sdpivot.go (SDPivotRouter + RegisterRoutes)
//
// The SDPivotRouter is injected into WeKnora's RouterParams via dig
// and routes are registered in NewRouter() automatically.
//
// JWT secret is configured via SDP_JWT_SECRET, with SMARTKNORA_JWT_SECRET fallback.
package main
