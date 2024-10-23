// Package extract provides a function to extract files from a reader to a destination.
//
// The extraction is done according to the file type, and the package provides a default extraction target OS.
// Available extractors are defined in the internal/extractor package. The package also provides an interface
// for the target that must be implemented to perform the unpacking process. This packages provides as well a
// memory target that provides an in-memory filesystem. The package also provides errors that can be returned
// during the unpacking process.
//
// Configuration is done using the [config] package, which provides a configuration struct that can be used to
// set the extraction type, the logger, the telemetry hook, and the maximum input size. Telemetry data is captured
// during the extraction process. The collection of telemetry data is done using the [telemetry] package.
package extract
