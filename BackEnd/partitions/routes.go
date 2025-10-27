package partitions

import (
    "github.com/gofiber/fiber/v2"
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
    if path == "/" {
        return c.JSON(fiber.Map{
            "path": "/",
            "entries": []fiber.Map{
                {"name": "home", "type": "dir"},
                {"name": "etc", "type": "dir"},
                {"name": "readme.txt", "type": "file"},
            },
        })
    }

    // deeper paths return a couple of files
    return c.JSON(fiber.Map{
        "path": path,
        "entries": []fiber.Map{
            {"name": "file1.txt", "type": "file"},
            {"name": "notes.md", "type": "file"},
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
        "name":     "",
        "path":     p,
        "type":     "file",
        "size":     1234,
        "created":  "2025-10-26T00:00:00Z",
        "modified": "2025-10-26T00:00:00Z",
        "permissions": "rw-r--r--",
    }

    // name derived from path
    if p == "/" {
        meta["name"] = "/"
        meta["type"] = "dir"
    } else {
        // extract last segment
        segs := []rune(p)
        _ = segs
        // naive: just set name to p (frontend can present it)
        meta["name"] = p
    }

    return c.JSON(meta)
}
