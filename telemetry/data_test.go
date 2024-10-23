package telemetry

import (
	"fmt"
	"testing"
	"time"
)

// TestDataString tests the String method of the data struct
func TestDataString(t *testing.T) {
	m := Data{
		ExtractedType:       "tar",
		ExtractionDuration:  time.Duration(5 * time.Millisecond),
		ExtractionSize:      1024,
		ExtractedFiles:      5,
		ExtractedSymlinks:   2,
		ExtractedDirs:       1,
		ExtractionErrors:    1,
		LastExtractionError: fmt.Errorf("example error"),
		InputSize:           2048,
		UnsupportedFiles:    0,
	}

	expected := `{"LastExtractionError":"example error","ExtractedDirs":1,"ExtractionDuration":5000000,"ExtractionErrors":1,"ExtractedFiles":5,"ExtractionSize":1024,"ExtractedSymlinks":2,"ExtractedType":"tar","InputSize":2048,"PatternMismatches":0,"UnsupportedFiles":0,"LastUnsupportedFile":""}`
	if m.String() != expected {
		t.Errorf("Expected '%s', but got '%s'", expected, m.String())
	}
}

func ExampleTelemetryHook() {

	// in, err := os.Open("archive.tar.gz")
	// if err != nil {
	// 	// handle error
	// }
	// defer in.Close()

	// hook := func(ctx context.Context, td *Data) {
	// 	// send td to a telemetry service, e.g., DataDog/SumoLogic
	// }

	// // setup the context and configuration
	// cfg := config.NewConfig(
	// 	config.WithTelemetryHook(hook),
	// )

	// // perform the extraction with the given configuration
	// ctx := context.Background()
	// dst := "destination"
	// if err := extract.Unpack(context.Background(), in, dst, cfg); err != nil {
	// 	// handle error
	// }
}
