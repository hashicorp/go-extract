package telemetry

import "context"

// NoopTelemetryHook is a no operation telemetry hook.
func NoopTelemetryHook(ctx context.Context, d *Data) {
	// noop
}
