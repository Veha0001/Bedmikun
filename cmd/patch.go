package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// idaPattern holds the bytes and masks for a pattern.
type idaPattern struct {
	pattern []byte
	mask    []byte
}

type patchSignature struct {
	find    idaPattern
	replace idaPattern
}

var signatures = map[string][]patchSignature{
	"x64": {
		// Signature 1: Changes a conditional jump related to trial mode.
		// Original: 15 B0 01 ... 48 89
		// Patched:  15 B0 00 ... 48 89
		{
			find:    parseIDAPattern("15 B0 01 48 8B 4C ?? ?? 48 33 ?? ?? ?? ?? ?? ?? 48 83 C4 40 5B C3 48 8B ?? ?? ?? ?? ?? 48 89"),
			replace: parseIDAPattern("15 B0 00 48 8B 4C ?? ?? 48 33 ?? ?? ?? ?? ?? ?? 48 83 C4 40 5B C3 48 8B ?? ?? ?? ?? ?? 48 89"),
		},
		// Signature 2: Another trial check bypass.
		// Original: 84 C0 74 ... B0 01
		// Patched:  84 C0 74 ... B0 00
		{
			find:    parseIDAPattern("84 C0 74 23 48 83 C3 10 48 3B DF 75 E3 B0 01 48"),
			replace: parseIDAPattern("84 C0 74 23 48 83 C3 10 48 3B DF 75 E3 B0 00 48"),
		},
	},
}

// parseIDAPattern converts a libhat-style pattern string into a byte pattern and a mask.
func parseIDAPattern(pattern string) idaPattern {
	parts := strings.Fields(pattern)
	pBytes := make([]byte, 0, len(parts))
	mBytes := make([]byte, 0, len(parts))

	for _, part := range parts {
		var p, m byte
		switch len(part) {
		case 1:
			if part == "?" {
				p, m = 0x00, 0x00
			}
		case 2:
			if part == "??" {
				p, m = 0x00, 0x00
			} else if part[0] == '?' { // ?B
				val, err := strconv.ParseUint(string(part[1]), 16, 4)
				if err == nil {
					p, m = byte(val), 0x0F
				}
			} else if part[1] == '?' { // A?
				val, err := strconv.ParseUint(string(part[0]), 16, 4)
				if err == nil {
					p, m = byte(val<<4), 0xF0
				}
			} else { // AB
				val, err := strconv.ParseUint(part, 16, 8)
				if err == nil {
					p, m = byte(val), 0xFF
				}
			}
		case 8: // Binary string
			maskStr := strings.Map(func(r rune) rune {
				if r == '?' {
					return '0'
				}
				return '1'
			}, part)
			valStr := strings.ReplaceAll(part, "?", "0")

			maskVal, err1 := strconv.ParseUint(maskStr, 2, 8)
			val, err2 := strconv.ParseUint(valStr, 2, 8)

			if err1 == nil && err2 == nil {
				p, m = byte(val&maskVal), byte(maskVal)
			}
		}
		pBytes = append(pBytes, p)
		mBytes = append(mBytes, m)
	}
	return idaPattern{pattern: pBytes, mask: mBytes}
}

// findPattern searches for a pattern in data using a mask.
func findPattern(data []byte, pat idaPattern) [][]int {
	var results [][]int
	pattern := pat.pattern
	mask := pat.mask
	patternLen := len(pattern)

	if patternLen == 0 {
		return results
	}

	for i := 0; i <= len(data)-patternLen; i++ {
		match := true
		for j := 0; j < patternLen; j++ {
			if (data[i+j] & mask[j]) != pattern[j] {
				match = false
				break
			}
		}
		if match {
			results = append(results, []int{i, i + patternLen})
		}
	}
	return results
}

// PatchFile finds and replaces byte signatures in the target file.
func PatchFile(targetPath string, backup bool) error {
	logger.Info("Starting patch process...", "file", targetPath)
	if backup {
		backupPath := targetPath + ".bak"
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			logger.Info("Creating backup...", "to", backupPath)
			if err := copyFile(targetPath, backupPath); err != nil {
				return err // copyFile already returns a descriptive error
			}
		} else {
			logger.Info("Backup file already exists, skipping creation.", "backup", backupPath)
		}
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return err
	}

	modified := false
	patchCount := 0
	sigs, ok := signatures["x64"] // Assuming x64 for now
	if !ok {
		return errors.New("no signatures found for architecture x64")
	}

	for i, sig := range sigs {
		locs := findPattern(data, sig.find)
		if len(locs) == 0 {
			logger.Warn(fmt.Sprintf("Not found: %X", sig.find.pattern), "signature_index", i+1)
			continue
		}

		logger.Debug("Signature found. Patching...", "signature_index", i+1, "occurrences", len(locs))

		for _, loc := range locs {
			start := loc[0]
			end := loc[1]
			originalSlice := data[start:end]

			logger.Debug(fmt.Sprintf("Found: %X", sig.find.pattern))
			logger.Debug(fmt.Sprintf("Replace: %X", sig.replace.pattern))

			replacementBytes := getReplacementBytes(sig.replace, originalSlice)
			if len(replacementBytes) != len(originalSlice) {
				logger.Error("Mismatch in pattern length, skipping patch for this occurrence to avoid corruption.", "signature_index", i+1)
				continue
			}
			copy(data[start:end], replacementBytes)
			logger.Info(fmt.Sprintf(" > Offset: 0x%X", start))
			patchCount++
		}
		modified = true
	}

	if !modified {
		logger.Warn("No signatures were found in the file. The file might already be patched or is an unsupported version.")
		return nil
	}

	if err := os.WriteFile(targetPath, data, 0666); err != nil {
		return err
	}

	logger.Info("Patching complete.", "total_patches", patchCount)
	return nil
}

// RestoreFile finds and reverts patched byte signatures in the target file.
func RestoreFile(targetPath string) error {
	logger.Info("Starting restore process...", "file", targetPath)

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return err
	}

	modified := false
	restoreCount := 0
	sigs, ok := signatures["x64"] // Assuming x64 for now
	if !ok {
		return errors.New("no signatures found for architecture x64")
	}

	for i, sig := range sigs {
		// In restore, we find the "replace" pattern and replace it with the "find" pattern.
		locs := findPattern(data, sig.replace)
		if len(locs) == 0 {
			// This is not a warning in restore, as some signatures might not have been applied.
			continue
		}

		logger.Debug("Found patched signature. Restoring...", "signature_index", i+1, "occurrences", len(locs))

		for _, loc := range locs {
			start := loc[0]
			end := loc[1]
			originalSlice := data[start:end]

			logger.Debug(fmt.Sprintf("Found: %X", sig.replace.pattern))
			logger.Debug(fmt.Sprintf("Replace: %X", sig.find.pattern))

			// We use sig.find to get the original bytes.
			replacementBytes := getReplacementBytes(sig.find, originalSlice)
			if len(replacementBytes) != len(originalSlice) {
				// This should ideally not happen if signatures are correct.
				logger.Error("Mismatch in pattern length, skipping restore for this occurrence to avoid corruption.", "signature_index", i+1)
				continue
			}
			copy(data[start:end], replacementBytes)
			logger.Info(fmt.Sprintf(" > Offset: 0x%X", start))
			restoreCount++
		}
		modified = true
	}

	if !modified {
		logger.Warn("No patched signatures were found in the file. The file might not be patched or is an unsupported version.")
		return nil
	}

	if err := os.WriteFile(targetPath, data, 0666); err != nil {
		return err
	}

	logger.Info("Restore complete.", "total_restored", restoreCount)
	return nil
}

// getReplacementBytes creates the byte slice for replacement, preserving wildcards from the original.
func getReplacementBytes(replace idaPattern, originalSlice []byte) []byte {
	result := make([]byte, len(replace.pattern))
	copy(result, replace.pattern) // Start with the replacement pattern

	// Apply wildcards: where mask is 0, use original byte
	for i := 0; i < len(result); i++ {
		if replace.mask[i] != 0xFF {
			// This handles full wildcards (0x00) and partials (e.g., 0xF0, 0x0F)
			// Keep original byte where replace mask is not full
			result[i] = (originalSlice[i] & ^replace.mask[i]) | (replace.pattern[i] & replace.mask[i])
		}
	}
	return result
}
