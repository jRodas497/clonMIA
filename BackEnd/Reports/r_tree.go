package Reports

import (
	"fmt"
	"os"
	"strings"

	Estructuras "backend/Estructuras"
)


// DirectoryTree representa el árbol de directorios (no JSON, solo uso interno)
type DirectoryTree struct {
	Name     string
	Children []*DirectoryTree
	IsDir    bool
}

// buildDirectoryTree construye recursivamente el árbol de directorios a partir del inodo indicado y el path actual.
func buildDirectoryTree(sb *Estructuras.SuperBlock, archivo *os.File, inodeIndex int32, currentPath string) (*DirectoryTree, error) {
	inodo, err := leerInodo(sb, archivo, inodeIndex)
	if err != nil {
		return nil, fmt.Errorf("error al leer inodo %d para '%s': %v", inodeIndex, currentPath, err)
	}
	var currentName string
	if currentPath == "/" {
		currentName = "/"
	} else {
		pathSegments := strings.Split(strings.Trim(currentPath, "/"), "/")
		currentName = pathSegments[len(pathSegments)-1]
	}
	tree := &DirectoryTree{
		Name:  currentName,
		IsDir: inodo.I_type[0] == '0',
	}
	if !tree.IsDir {
		return tree, nil
	}
	for _, blockIndex := range inodo.I_block {
		if blockIndex == -1 {
			break
		}
		bloque := &Estructuras.FolderBlock{}
		offset := int64(sb.S_block_start + blockIndex*sb.S_block_size)
		err := bloque.Decodificar(archivo, offset)
		if err != nil {
			return nil, fmt.Errorf("error al deserializar el bloque %d: %v", blockIndex, err)
		}
		for _, content := range bloque.B_cont {
			if content.B_inodo == -1 {
				continue
			}
			nombre := strings.Trim(string(content.B_name[:]), "\x00 ")
			if nombre == "." || nombre == ".." {
				continue
			}
			childPath := currentPath + "/" + nombre
			childNode, err := buildDirectoryTree(sb, archivo, content.B_inodo, childPath)
			if err != nil {
				return nil, fmt.Errorf("error al construir el árbol para '%s': %v", childPath, err)
			}
			tree.Children = append(tree.Children, childNode)
		}
	}
	return tree, nil
}

// generateDirectoryTreeDot genera el DOT del árbol de directorios
func generateDirectoryTreeDot(sb *Estructuras.SuperBlock, archivo *os.File) (string, error) {
	tree, err := buildDirectoryTree(sb, archivo, 0, "/")
	if err != nil {
		return "", err
	}
	const header = `digraph DirectoryTree {
	rankdir=TB;
	node [shape=box, style="rounded,filled", fontname="Helvetica"];
	edge [arrowhead=vee, color="#555555"];
`
	const footer = `
}
`
	var lines []string
	lines = append(lines, header)
	nodeCounter := 0
	nodeIDs := make(map[*DirectoryTree]string)
	var buildDot func(node *DirectoryTree, parentID string, depth int)
	buildDot = func(node *DirectoryTree, parentID string, depth int) {
		id := fmt.Sprintf("node%d", nodeCounter)
		nodeCounter++
		nodeIDs[node] = id
		var fill, font, border string
		if node.IsDir {
			fill = "#4285F4"
			font = "#FFFFFF"
			border = "#2B579A"
		} else {
			fill = "#34A853"
			font = "#FFFFFF"
			border = "#333333"
		}
		shape := "note"
		label := node.Name
		if node.IsDir {
			shape = "folder"
			if label == "/" {
				label = "ROOT"
			}
		}
		lines = append(lines, fmt.Sprintf(
			"    %s [label=%q fillcolor=%q fontcolor=%q color=%q shape=%s];",
			id, label, fill, font, border, shape,
		))
		if parentID != "" {
			lines = append(lines, fmt.Sprintf("    %s -> %s;", parentID, id))
		}
		for _, c := range node.Children {
			buildDot(c, id, depth+1)
		}
	}
	buildDot(tree, "", 0)
	lines = append(lines, footer)
	return strings.Join(lines, "\n"), nil
}

// ReporteTree genera el reporte del árbol ext2 mostrando toda la información de inodos y bloques
func ReporteTree(sb *Estructuras.SuperBlock, rutaDisco string, ruta string) error {
	err := os.MkdirAll(getParentDir(ruta), 0755)
	if err != nil {
		return fmt.Errorf("error al crear directorios: %v", err)
	}
	archivo, err := os.Open(rutaDisco)
	if err != nil {
		return fmt.Errorf("error al abrir el disco: %v", err)
	}
	defer archivo.Close()

	dot, err := generateDirectoryTreeDot(sb, archivo)
	if err != nil {
		return err
	}
	err = escribirArchivoDot(ruta, dot)
	if err != nil {
		return err
	}
	fmt.Println("Reporte TREE generado:", ruta)
	return nil
}

// getParentDir obtiene el directorio padre de una ruta
func getParentDir(path string) string {
	idx := strings.LastIndex(path, string(os.PathSeparator))
	if idx == -1 {
		return "."
	}
	return path[:idx]
}