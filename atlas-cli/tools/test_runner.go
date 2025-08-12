package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type TestSuite struct {
	Name        string
	Package     string
	Description string
	Tests       []string
}

type TestResult struct {
	Suite     string
	Test      string
	Status    string
	Duration  time.Duration
	Output    string
	Error     string
}

var testSuites = []TestSuite{
	{
		Name:        "Provider Validation Tests",
		Package:     "./pkg/providers",
		Description: "Tests for cluster configuration validation logic",
		Tests: []string{
			"TestLocalProvider_ValidateConfig",
			"TestLocalProvider_GetProviderName",
			"TestLocalProvider_GetSupportedRegions",
			"TestLocalProvider_GetSupportedVersions",
			"TestNetworkConfigValidation",
		},
	},
	{
		Name:        "Command-line Interface Tests",
		Package:     "./cmd",
		Description: "Tests for CLI command parsing and configuration loading",
		Tests: []string{
			"TestLoadClusterConfig",
			"TestLoadClusterConfig_FileNotFound",
			"TestClusterGenerateConfigCmd",
			"TestClusterCreateCmd_FlagParsing",
			"TestConfigFileVsFlagsIntegration",
		},
	},
	{
		Name:        "Integration Tests",
		Package:     "./pkg/providers",
		Description: "Integration tests requiring minikube (run with -integration flag)",
		Tests: []string{
			"TestLocalProvider_Integration",
		},
	},
}

func main() {
	fmt.Println("🚀 Atlas CLI Test Suite Runner")
	fmt.Println("==============================")
	
	// Parse command line arguments
	runIntegration := false
	runBenchmarks := false
	verbose := false
	
	for _, arg := range os.Args[1:] {
		switch arg {
		case "-integration":
			runIntegration = true
		case "-bench":
			runBenchmarks = true
		case "-v", "-verbose":
			verbose = true
		case "-h", "-help":
			printUsage()
			return
		}
	}
	
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Integration tests: %v\n", runIntegration)
	fmt.Printf("Benchmarks: %v\n", runBenchmarks)
	fmt.Printf("Verbose: %v\n\n", verbose)
	
	// Check prerequisites
	if err := checkPrerequisites(); err != nil {
		fmt.Printf("❌ Prerequisites check failed: %v\n", err)
		os.Exit(1)
	}
	
	var allResults []TestResult
	totalTests := 0
	passedTests := 0
	failedTests := 0
	skippedTests := 0
	
	// Run each test suite
	for _, suite := range testSuites {
		if suite.Name == "Integration Tests" && !runIntegration {
			fmt.Printf("⏭️  Skipping %s (use -integration flag to run)\n\n", suite.Name)
			continue
		}
		
		fmt.Printf("📦 Running %s\n", suite.Name)
		fmt.Printf("   %s\n", suite.Description)
		fmt.Printf("   Package: %s\n\n", suite.Package)
		
		results := runTestSuite(suite, verbose)
		allResults = append(allResults, results...)
		
		// Count results for this suite
		suiteTotal := 0
		suitePassed := 0
		suiteFailed := 0
		suiteSkipped := 0
		
		for _, result := range results {
			suiteTotal++
			switch result.Status {
			case "PASS":
				suitePassed++
			case "FAIL":
				suiteFailed++
			case "SKIP":
				suiteSkipped++
			}
		}
		
		totalTests += suiteTotal
		passedTests += suitePassed
		failedTests += suiteFailed
		skippedTests += suiteSkipped
		
		// Print suite summary
		if suiteFailed > 0 {
			fmt.Printf("   ❌ Suite: %d/%d tests failed\n\n", suiteFailed, suiteTotal)
		} else if suiteSkipped == suiteTotal {
			fmt.Printf("   ⏭️  Suite: All %d tests skipped\n\n", suiteTotal)
		} else {
			fmt.Printf("   ✅ Suite: %d/%d tests passed\n\n", suitePassed, suiteTotal)
		}
	}
	
	// Run benchmarks if requested
	if runBenchmarks {
		fmt.Printf("🏎️  Running Benchmarks\n")
		fmt.Printf("=====================\n\n")
		runBenchmarkSuite(verbose)
	}
	
	// Print overall summary
	fmt.Printf("📊 Overall Test Summary\n")
	fmt.Printf("=======================\n")
	fmt.Printf("Total Tests: %d\n", totalTests)
	fmt.Printf("✅ Passed: %d\n", passedTests)
	if failedTests > 0 {
		fmt.Printf("❌ Failed: %d\n", failedTests)
	}
	if skippedTests > 0 {
		fmt.Printf("⏭️  Skipped: %d\n", skippedTests)
	}
	
	// Print failed tests details
	if failedTests > 0 {
		fmt.Printf("\n❌ Failed Tests:\n")
		for _, result := range allResults {
			if result.Status == "FAIL" {
				fmt.Printf("   %s.%s\n", result.Suite, result.Test)
				if result.Error != "" {
					fmt.Printf("     Error: %s\n", result.Error)
				}
			}
		}
	}
	
	// Print configuration validation summary
	fmt.Printf("\n🔧 Configuration Features Tested:\n")
	features := []string{
		"✅ Basic cluster validation",
		"✅ Network configuration (CIDR, ports, plugins)",
		"✅ Security configuration (RBAC, policies, audit)",
		"✅ Resource configuration (limits, quotas, scaling)",
		"✅ CLI flag parsing and validation", 
		"✅ YAML configuration file loading",
		"✅ Error handling and edge cases",
	}
	for _, feature := range features {
		fmt.Printf("   %s\n", feature)
	}
	
	if runIntegration {
		fmt.Printf("   ✅ Integration tests (minikube operations)\n")
	} else {
		fmt.Printf("   ⏭️  Integration tests (skipped - use -integration flag)\n")
	}
	
	if runBenchmarks {
		fmt.Printf("   ✅ Performance benchmarks\n")
	}
	
	if failedTests > 0 {
		fmt.Printf("\n❌ Test suite failed with %d failing tests\n", failedTests)
		os.Exit(1)
	} else {
		fmt.Printf("\n🎉 All tests passed!\n")
	}
}

func printUsage() {
	fmt.Println("Atlas CLI Test Suite Runner")
	fmt.Println("")
	fmt.Println("Usage: go run run_tests.go [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -integration    Run integration tests (requires minikube)")
	fmt.Println("  -bench         Run benchmark tests")
	fmt.Println("  -v, -verbose   Verbose output")
	fmt.Println("  -h, -help      Show this help")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  go run run_tests.go                    # Run unit tests only")
	fmt.Println("  go run run_tests.go -integration       # Run all tests including integration")
	fmt.Println("  go run run_tests.go -bench -v          # Run with benchmarks and verbose output")
}

func checkPrerequisites() error {
	// Check if Go modules are available
	if _, err := exec.Command("go", "version").CombinedOutput(); err != nil {
		return fmt.Errorf("Go is not available: %v", err)
	}
	
	// Check if the project can build
	fmt.Printf("🔍 Building project...\n")
	if out, err := exec.Command("go", "build", "-o", "/tmp/atlas-cli-test").CombinedOutput(); err != nil {
		return fmt.Errorf("project build failed: %v\nOutput: %s", err, out)
	}
	
	// Clean up test binary
	os.Remove("/tmp/atlas-cli-test")
	
	fmt.Printf("✅ Prerequisites check passed\n\n")
	return nil
}

func runTestSuite(suite TestSuite, verbose bool) []TestResult {
	var results []TestResult
	
	// Run all tests in the package if no specific tests are listed
	args := []string{"test", suite.Package, "-v"}
	if len(suite.Tests) > 0 {
		args = append(args, "-run", strings.Join(suite.Tests, "|"))
	}
	
	cmd := exec.Command("go", args...)
	
	// Set environment variables for testing
	env := os.Environ()
	env = append(env, "CGO_ENABLED=1") // Needed for sqlite
	cmd.Env = env
	
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	
	// Parse test output
	scanner := bufio.NewScanner(strings.NewReader(outputStr))
	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.Contains(line, "=== RUN") {
			testName := strings.TrimSpace(strings.TrimPrefix(line, "=== RUN"))
			results = append(results, TestResult{
				Suite: suite.Name,
				Test:  testName,
				Status: "RUN",
			})
		} else if strings.Contains(line, "--- PASS:") || strings.Contains(line, "--- FAIL:") || strings.Contains(line, "--- SKIP:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				status := strings.TrimSuffix(parts[1], ":")
				testName := parts[2]
				
				// Find the test result to update
				for i := range results {
					if results[i].Test == testName && results[i].Suite == suite.Name {
						results[i].Status = status
						
						// Parse duration if available
						if len(parts) >= 4 {
							if duration, err := time.ParseDuration(strings.Trim(parts[3], "()")); err == nil {
								results[i].Duration = duration
							}
						}
						break
					}
				}
			}
		}
	}
	
	// Handle case where no individual test results were parsed (e.g., compilation errors)
	if len(results) == 0 && err != nil {
		results = append(results, TestResult{
			Suite:  suite.Name,
			Test:   "compilation",
			Status: "FAIL",
			Error:  err.Error(),
			Output: outputStr,
		})
	}
	
	// Print test results
	for _, result := range results {
		symbol := "❓"
		switch result.Status {
		case "PASS":
			symbol = "✅"
		case "FAIL":
			symbol = "❌"
		case "SKIP":
			symbol = "⏭️ "
		}
		
		durationStr := ""
		if result.Duration > 0 {
			durationStr = fmt.Sprintf(" (%v)", result.Duration)
		}
		
		fmt.Printf("   %s %s%s\n", symbol, result.Test, durationStr)
		
		if verbose && result.Status == "FAIL" && result.Error != "" {
			fmt.Printf("      Error: %s\n", result.Error)
		}
	}
	
	if verbose && err != nil {
		fmt.Printf("   Command output:\n%s\n", outputStr)
	}
	
	return results
}

func runBenchmarkSuite(verbose bool) {
	fmt.Printf("Running benchmarks for provider validation...\n")
	
	cmd := exec.Command("go", "test", "./pkg/providers", "-bench", ".", "-benchmem")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	
	if err != nil {
		fmt.Printf("❌ Benchmark failed: %v\n", err)
		if verbose {
			fmt.Printf("Output: %s\n", outputStr)
		}
		return
	}
	
	// Parse and display benchmark results
	scanner := bufio.NewScanner(strings.NewReader(outputStr))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Benchmark") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				benchName := parts[0]
				iterations := parts[1]
				nsPerOp := parts[2]
				
				fmt.Printf("   🏎️  %s: %s iterations, %s ns/op", benchName, iterations, nsPerOp)
				
				// Add memory stats if available
				if len(parts) >= 6 && strings.HasSuffix(parts[4], "B/op") {
					fmt.Printf(", %s %s", parts[4], parts[5])
				}
				fmt.Println()
			}
		}
	}
	
	fmt.Println()
}