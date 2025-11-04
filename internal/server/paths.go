package server

import (
	"path/filepath"
	"strings"
)

func paths(root, fileID, fileName string) (tmpDir, finalPath, tempFinal string) {
	safe := sanitizeFilename(filepath.Base(fileName))
	tmpDir = filepath.Join(root, "tmp", fileID)
	finalPath = filepath.Join(root, "files", fileID+"_"+safe)
	tempFinal = finalPath + ".part"
	return
}

func sanitizeFilename(name string) string {
	// Remove dangerous characters
	dangerous := []string{"..", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range dangerous {
		name = strings.ReplaceAll(name, char, "_")
	}
	// Ensure not empty
	if name == "" || name == "." {
		name = "file"
	}
	return name
}