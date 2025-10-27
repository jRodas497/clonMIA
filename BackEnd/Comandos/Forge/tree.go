package Forge

import (
    "fmt"
    "os"
    "strings"

    Estructuras "backend/Estructuras"
    Global "backend/Global"
    Utils "backend/Utils"
)

type DirectoryTree struct {
    Name     string           `json:"name"`
    Children []*DirectoryTree `json:"children,omitempty"`
    IsDir    bool             `json:"isDir"`
}

// DirectoryTreeService maneja la construcción y obtención del árbol de directorios.
type DirectoryTreeService struct {
    partitionSuperblock *Estructuras.SuperBlock
    partitionPath       string
    file                *os.File
}

func NewDirectoryTreeService() (*DirectoryTreeService, error) {
    // Verificar sesión
    if !Global.VerificarSesionActiva() {
        return nil, fmt.Errorf("error: no hay un usuario logueado")
    }

    idPartition := Global.UsuarioActual.Id
    partitionSuperblock, _, partitionPath, err := Global.ObtenerSuperblockParticionMontada(idPartition)
    if err != nil {
        return nil, fmt.Errorf("error al obtener la partición montada (ID: %s): %w", idPartition, err)
    }

    file, err := os.OpenFile(partitionPath, os.O_RDWR, 0666)
    if err != nil {
        return nil, fmt.Errorf("error al abrir el archivo de la partición en '%s': %w", partitionPath, err)
    }

    return &DirectoryTreeService{
        partitionSuperblock: partitionSuperblock,
        partitionPath:       partitionPath,
        file:                file,
    }, nil
}

func (dts *DirectoryTreeService) Close() {
    if dts.file != nil {
        dts.file.Close()
    }
}

func (dts *DirectoryTreeService) GetDirectoryTree(path string) (*DirectoryTree, error) {
    var rootInodeIndex int32
    var err error

    if path == "/" {
        rootInodeIndex = 0
    } else {
        parentDirs, dirName := Utils.ObtenerDirectoriosPadre(path)
        rootInodeIndex, err = buscarInodoArchivo(dts.file, dts.partitionSuperblock, parentDirs, dirName)
        if err != nil {
            return nil, fmt.Errorf("error al encontrar el directorio inicial '%s': %w", path, err)
        }
    }

    tree, err := dts.buildDirectoryTree(rootInodeIndex, path)
    if err != nil {
        return nil, fmt.Errorf("error al construir el árbol de directorios para '%s': %w", path, err)
    }

    return tree, nil
}

// buildDirectoryTree construye recursivamente el árbol de directorios a partir del inodo indicado y el path actual.
func (dts *DirectoryTreeService) buildDirectoryTree(inodeIndex int32, currentPath string) (*DirectoryTree, error) {
    inodo := &Estructuras.INodo{}
    offset := int64(dts.partitionSuperblock.S_inode_start) + int64(inodeIndex*dts.partitionSuperblock.S_inode_size)
    if err := inodo.Decodificar(dts.file, offset); err != nil {
        return nil, fmt.Errorf("error al deserializar el inodo %d (offset %d) para '%s': %w", inodeIndex, offset, currentPath, err)
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
        blockOffset := int64(dts.partitionSuperblock.S_block_start) + int64(blockIndex*dts.partitionSuperblock.S_block_size)
        if err := bloque.Decodificar(dts.file, blockOffset); err != nil {
            return nil, fmt.Errorf("error al deserializar el bloque %d (offset %d): %w", blockIndex, blockOffset, err)
        }

        for _, contenido := range bloque.B_cont {
            if contenido.B_inodo == -1 {
                continue
            }
            nombre := strings.Trim(string(contenido.B_name[:]), "\x00 ")
            if nombre == "." || nombre == ".." {
                continue
            }

            var childPath string
            if currentPath == "/" {
                childPath = "/" + nombre
            } else {
                childPath = currentPath + "/" + nombre
            }

            childNode, err := dts.buildDirectoryTree(contenido.B_inodo, childPath)
            if err != nil {
                return nil, fmt.Errorf("error al construir el árbol para '%s': %w", childPath, err)
            }
            tree.Children = append(tree.Children, childNode)
        }
    }

    return tree, nil
}

func (dts *DirectoryTreeService) GenerateDotGraph() (string, error) {
    // 1) Obtener el árbol desde la raíz
    tree, err := dts.GetDirectoryTree("/")
    if err != nil {
        return "", fmt.Errorf("error al obtener el árbol de directorios: %w", err)
    }

    // 2) Encabezado con configuración vertical (Top to Bottom)
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
