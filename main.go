package main

import (
	"archive/zip"
	"debug/elf"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: is16kbReady <path-to-apk>")
		os.Exit(1)
	}

	apkPath := os.Args[1]
	if !strings.HasSuffix(apkPath, ".apk") {
		fmt.Println("Error: File must be an APK")
		os.Exit(1)
	}

	if err := checkAPKAlignment(apkPath); err != nil {
		log.Fatal(err)
	}
}

func checkAPKAlignment(apkPath string) error {
	r, err := zip.OpenReader(apkPath)
	if err != nil {
		return fmt.Errorf("failed to open APK: %v", err)
	}
	defer r.Close()

	var alignedLibs []string
	var unalignedLibs []string
	var errorLibs []string

	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "lib/") || !strings.HasSuffix(f.Name, ".so") {
			continue
		}

		if !strings.Contains(f.Name, "arm64-v8a") && !strings.Contains(f.Name, "x86_64") {
			continue
		}

		aligned, alignment, err := checkELFAlignment(f)
		if err != nil {
			errorLibs = append(errorLibs, fmt.Sprintf("%s: ERROR (%v)", f.Name, err))
			continue
		}

		if aligned {
			alignedLibs = append(alignedLibs, fmt.Sprintf("%s (%s)", f.Name, alignment))
		} else {
			unalignedLibs = append(unalignedLibs, fmt.Sprintf("%s (%s)", f.Name, alignment))
		}
	}

	printResults(alignedLibs, unalignedLibs, errorLibs)

	return nil
}

func printResults(alignedLibs, unalignedLibs, errorLibs []string) {
	totalLibs := len(alignedLibs) + len(unalignedLibs)

	if len(alignedLibs) > 0 {
		fmt.Printf("\n\033[32m✓ ALIGNED LIBRARIES (%d):\033[0m\n", len(alignedLibs))
		for _, lib := range alignedLibs {
			fmt.Printf("  %s\n", lib)
		}
	}

	if len(unalignedLibs) > 0 {
		fmt.Printf("\n\033[31m✗ UNALIGNED LIBRARIES (%d):\033[0m\n", len(unalignedLibs))
		for _, lib := range unalignedLibs {
			fmt.Printf("  %s\n", lib)
		}
	}

	if len(errorLibs) > 0 {
		fmt.Printf("\n\033[33m⚠ ERRORS (%d):\033[0m\n", len(errorLibs))
		for _, lib := range errorLibs {
			fmt.Printf("  %s\n", lib)
		}
	}

	fmt.Println()
	if len(unalignedLibs) > 0 {
		fmt.Printf("\033[31m❌ 16KB ALIGNMENT: NOT SUPPORTED\033[0m\n")
		fmt.Printf("   %d unaligned libs out of %d total libs need to be fixed.\n", len(unalignedLibs), totalLibs)
		fmt.Printf("   This app is NOT ready for 16KB page size devices.\n")
	} else if totalLibs > 0 {
		fmt.Printf("\033[32m✅ 16KB ALIGNMENT: SUPPORTED\033[0m\n")
		fmt.Printf("   All %d libraries are properly aligned for 16KB pages.\n", totalLibs)
		fmt.Printf("   This app is ready for 16KB page size devices.\n")
	} else {
		fmt.Printf("\033[33m⚠️  16KB ALIGNMENT: UNKNOWN\033[0m\n")
		fmt.Printf("   No ARM64/x86_64 libraries found in APK.\n")
		fmt.Printf("   Unable to determine 16KB page size compatibility.\n")
	}
}

func checkELFAlignment(f *zip.File) (bool, string, error) {
	rc, err := f.Open()
	if err != nil {
		return false, "", err
	}
	defer rc.Close()

	tmpFile, err := os.CreateTemp("", "lib_*.so")
	if err != nil {
		return false, "", err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, rc); err != nil {
		return false, "", err
	}

	if err := tmpFile.Close(); err != nil {
		return false, "", err
	}

	elfFile, err := elf.Open(tmpFile.Name())
	if err != nil {
		return false, "", fmt.Errorf("not an ELF file")
	}
	defer elfFile.Close()

	for _, prog := range elfFile.Progs {
		if prog.Type == elf.PT_LOAD {
			align := prog.Align
			alignStr := fmt.Sprintf("2**%d", getLog2(align))

			if align >= 16384 {
				return true, alignStr, nil
			} else {
				return false, alignStr, nil
			}
		}
	}

	return false, "no LOAD segment", nil
}

func getLog2(n uint64) int {
	if n == 0 {
		return 0
	}
	log := 0
	for n > 1 {
		n >>= 1
		log++
	}
	return log
}
