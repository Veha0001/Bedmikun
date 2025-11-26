package cmd

import (
	"bytes"
	"debug/pe"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// PatchInstruction defines a single patching operation.
type PatchInstruction struct {
	SearchPatternHex string // The pattern to find in the executable as a hex string
	FindBytes        []byte // The bytes to find within the SearchPattern
	ReplaceBytes     []byte // The bytes to replace FindBytes with
}

// parseHexString converts a hex string with optional '??' wildcards into a byte slice.
func parseHexString(hexString string) ([]byte, error) {
	parts := strings.Fields(hexString)
	var result []byte
	for _, part := range parts {
		if part == "??" {
			result = append(result, '?') // Use '?' as our wildcard byte
		} else {
			b, err := hex.DecodeString(part)
			if err != nil {
				return nil, fmt.Errorf("invalid hex string part: %s, %w", part, err)
			}
			result = append(result, b...)
		}
	}
	return result, nil
}


// getArchitecture detects the architecture of the PE file.
func getArchitecture(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	peFile, err := pe.NewFile(file)
	if err != nil {
		return "", fmt.Errorf("failed to parse PE file: %w", err)
	}
	defer peFile.Close()

	switch peFile.FileHeader.Machine {
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return "x64", nil
	case pe.IMAGE_FILE_MACHINE_I386:
		return "x86", nil
	default:
		return "unknown", nil
	}
}

// PatchFile creates a backup of the original file, then patches it.
func PatchFile(filePath string) error {
	arch, err := getArchitecture(filePath)
	if err != nil {
		return fmt.Errorf("failed to detect architecture: %w", err)
	}
	logger.Infof("Detected binary architecture: %s", arch)

	// Define all patching instructions.
	// The SearchPattern corresponds to the full signature including B0 01.
	// FindBytes is the sequence to locate *within* the found SearchPattern.
	// ReplaceBytes is what FindBytes will be changed to.
	allPatchInstructions := map[string][]PatchInstruction{
		"x64": {
			{
				SearchPattern: func() []byte {
					p, err := parseHexString("10 84 ?? ?? 15 B0 01 48 8B 4C ?? ?? 48 33 ?? ?? ?? ?? ?? ?? 48 83 C4 40 5B C3 48 8B ?? ?? ?? ?? 48 89")
					if err != nil {
						panic(fmt.Sprintf("failed to parse x64 sig1: %v", err))
					}
					return p
				}(),
				FindBytes:    []byte{0xB0, 0x01},
				ReplaceBytes: []byte{0xB0, 0x00},
			},
			{
				SearchPattern: func() []byte {
					p, err := parseHexString("84 C0 74 23 48 83 C3 10 48 3B DF 75 E3 B0 01 48")
					if err != nil {
						panic(fmt.Sprintf("failed to parse x64 sig2: %v", err))
					}
					return p
				}(),
				FindBytes:    []byte{0xB0, 0x01},
				ReplaceBytes: []byte{0xB0, 0x00},
			},
		},
	}

	patchInstructions, ok := allPatchInstructions[arch]
	if !ok || len(patchInstructions) == 0 {
		return fmt.Errorf("no patching instructions found for %s architecture", arch)
	}

	backupPath := filePath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		// create backup if it doesn't exist
		originalData, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read original file for backup: %w", err)
		}
		if err := os.WriteFile(backupPath, originalData, 0644); err != nil {
			return fmt.Errorf("failed to write backup file: %w", err)
		}
		logger.Infof("Created backup at %s", backupPath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	patched := false
	for _, instr := range patchInstructions {
		offset := findSignature(content, instr.SearchPattern)
		if offset != -1 {
			logger.Infof("Found pattern at offset 0x%X", offset)
			// Search for FindBytes within the found pattern's region
			findOffset := bytes.Index(content[offset:offset+len(instr.SearchPattern)], instr.FindBytes)
			if findOffset != -1 {
				// Apply replacement
				// The +1 is to target the byte after B0, which is the 01 we want to change to 00
				// This assumes FindBytes is always 2 bytes and we want to change the second byte.
				// If FindBytes can be of arbitrary length and we want to replace the whole sequence,
				// the logic here needs to be more generic. For B0 01 -> B0 00, this is fine.
				if len(instr.FindBytes) == 2 && len(instr.ReplaceBytes) == 2 {
					content[offset+findOffset+1] = instr.ReplaceBytes[1]
					logger.Infof("Patched pattern at offset 0x%X (original find offset 0x%X)", offset+findOffset, findOffset)
					patched = true
				} else {
					logger.Warnf("Patch instruction for pattern at 0x%X has FindBytes/ReplaceBytes of unexpected length. FindBytes: %d, ReplaceBytes: %d", offset, len(instr.FindBytes), len(instr.ReplaceBytes))
				}
			}
		}
	}

	if !patched {
		logger.Warn("Could not find any of the signatures to patch.")
		return nil
	}

	return os.WriteFile(filePath, content, 0644)
}

// RestoreFile restores the executable from a backup.
func RestoreFile(filePath string) error {
	backupPath := filePath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup file found at %s", backupPath)
	}

	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	if err := os.WriteFile(filePath, backupData, 0644); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}
	logger.Infof("Restored %s from %s", filePath, backupPath)
	return nil
}

func findSignature(data []byte, signature []byte) int {
	for i := 0; i <= len(data)-len(signature); i++ {
		found := true
		for j := 0; j < len(signature); j++ {
			if signature[j] != '?' && data[i+j] != signature[j] {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	return -1
}
