package Disk

import (
	Global "backend/Global"
	"bytes"
	"fmt"
)

func ParserMounted(tokens []string) (string, error) {
	var bufferSalida bytes.Buffer

	if len(Global.ParticionesMontadas) == 0 {
		return "", fmt.Errorf("no hay particiones montadas actualmente")
	}

	fmt.Fprintln(&bufferSalida, "------------------------ Particiones Montadas ------------------------")
	for id, path := range Global.ParticionesMontadas {
		fmt.Fprintf(&bufferSalida, "ID: %s | Path: %s\n", id, path)
	}
	fmt.Fprint(&bufferSalida, "--------------------------------------------\n")
	return bufferSalida.String(), nil
}
