// Package extract provides a function to extract files from a reader to a destination.
//
// The extraction process is determined by the file type, with support for various formats
// that can be output to the underlying OS, in-memory, or a custom filesystem target.
//
// Configuration is done using the [Config], which is a configuration struct that can be used to
// set the extraction type, the logger, the telemetry hook, and the maximum input size. Telemetry data is captured
// during the extraction process. The collection of [TelemetryData] is done using the telemetry package.
package extract
