package partitions

import (
    "github.com/gofiber/fiber/v2"
    "os"
    "path/filepath"
    "strings"
    "os/user"
)

// RegisterRoutes registers partition-related endpoints for testing and development.
func RegisterRoutes(app *fiber.App) {
    app.Post("/api/disk/partition/list", listHandler)
    app.Post("/api/disk/partition/stat", statHandler)
}

type listReq struct {
    DiskPath      string `json:"diskPath"`
    PartitionName string `json:"partitionName"`
    Path          string `json:"path"`
}

type statReq = listReq

// listHandler returns a simple, deterministic list of entries for the requested path.
// This is a lightweight stub useful for frontend testing.
func listHandler(c *fiber.Ctx) error {
    var req listReq
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
    }

    // Normalize path
    path := req.Path
    if path == "" {
        path = "/"
    }

    // Simple stubbed content based on path
    // If the client requests the host filesystem (local server FS), serve real OS entries
    if req.DiskPath == "__hostfs" {
        clean := filepath.Clean(path)
        if !filepath.IsAbs(clean) {
            clean = "/"
        }

        files, err := os.ReadDir(clean)
        if err != nil {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot read path", "detail": err.Error()})
        }

        entries := make([]fiber.Map, 0, len(files))
        for _, fi := range files {
            name := fi.Name()
            info, _ := fi.Info()
            if fi.IsDir() {
                entries = append(entries, fiber.Map{"name": name, "type": "dir", "tipo": "carpeta", "extension": nil, "size": 0})
            } else {
                ext := strings.TrimPrefix(filepath.Ext(name), ".")
                var extVal interface{} = nil
                if ext != "" {
                    extVal = ext
                }
                size := int64(0)
                if info != nil {
                    size = info.Size()
                }
                entries = append(entries, fiber.Map{"name": name, "type": "file", "tipo": "file", "extension": extVal, "size": size})
            }
        }

        resp := fiber.Map{"path": clean, "entries": entries}
        // If root requested, provide autoHome to let the frontend auto-navigate to user's home
        if clean == "/" {
            if u, err := user.Current(); err == nil {
                resp["autoHome"] = u.HomeDir
            }
        }

        return c.JSON(resp)
    }

    // Simple stubbed content based on path
    if path == "/" {
        return c.JSON(fiber.Map{
            "path": "/",
            "entries": []fiber.Map{
                {"name": "home", "type": "dir", "tipo": "carpeta", "extension": nil, "size": 0},
                {"name": "etc", "type": "dir", "tipo": "carpeta", "extension": nil, "size": 0},
                {"name": "readme.txt", "type": "file", "tipo": "file", "extension": "txt", "size": 1024},
            },
        })
    }

    // deeper paths return a couple of files
    return c.JSON(fiber.Map{
        "path": path,
        "entries": []fiber.Map{
            {"name": "file1.txt", "type": "file", "tipo": "file", "extension": "txt", "size": 2048},
            {"name": "notes.md", "type": "file", "tipo": "file", "extension": "md", "size": 512},
        },
    })
}

// statHandler returns fake metadata for the requested path.
func statHandler(c *fiber.Ctx) error {
    var req statReq
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
    }

    p := req.Path
    if p == "" {
        p = "/"
    }

    // stubbed metadata
    meta := fiber.Map{
        "name":       "",
        "path":       p,
        "type":       "file",
        "tipo":       "file",
        "extension":  nil,
        "size":       1234,
        "created":    "2025-10-26T00:00:00Z",
        "modified":   "2025-10-26T00:00:00Z",
        "permissions": "rw-r--r--",
    }

    // name derived from path
    if p == "/" {
        meta["name"] = "/"
        meta["type"] = "dir"
        meta["tipo"] = "carpeta"
        meta["extension"] = nil
    } else {
        // extract last segment
        segs := []rune(p)
        _ = segs
        // naive: just set name to p (frontend can present it)
        meta["name"] = p
    }

    return c.JSON(meta)
}
