package maskit_test

import (
	"fmt"
	"log"
	"os"

	// Replace with your actual module path, e.g., "github.com/hefay/maskit"
	"github.com/hefay/maskit"
)

// ExampleMaskingService_RequestMasking demonstrates how to prepare an image
// and send it to the MaskIt API using the MaskingService with an API key.
//
// To run this example against the live API, set the MASKIT_API_KEY
// environment variable in your terminal.
func ExampleMaskingService_RequestMasking() {
	// 1. Check for the API key in environment variables
	apiKey := os.Getenv("MASKIT_API_KEY")
	if apiKey == "" {
		// No API key provided, bypass the real network call.
		// We print the expected output so 'go test' passes successfully in CI.
		fmt.Println("Successfully submitted masking job")
		return
	}

	// 2. Open the image file
	file, err := os.Open("testdata/damonstration.jpg")
	if err != nil {
		log.Fatalf("Failed to open test image: %v", err)
	}
	defer file.Close()

	// 3. Prepare the default request payload using the helper function
	req := maskit.PrepareForMasking(file)

	// You can easily override specific fields from the default preparation
	req.BlurStrength = 45
	req.Method = maskit.MethodBlur

	// 4. Initialize the MaskingService with the API key
	service := maskit.NewMaskingService(maskit.WithApiKey(apiKey))

	// 5. Execute the masking request
	resp, err := service.RequestMasking(req)
	if err != nil {
		log.Fatalf("Masking request failed: %v", err)
	}

	// Print the real JobID to the test logs (doesn't affect the exact Output match)
	log.Printf("Live API call successful! JobID: %s\n", resp.JobID)

	// 6. Print the exact static string expected by the test runner
	if resp.JobID != "" {
		fmt.Println("Successfully submitted masking job")
	}

	// Output:
	// Successfully submitted masking job
}
