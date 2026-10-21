package companion

import "github.com/gofiber/fiber/v3"

// CacheControlNoStore is the Cache-Control of every member-facing companion response (T-M9-02, FAM-L6): what a
// member reads about the owner must never sit in a shared or on-disk cache.
const CacheControlNoStore = "private, no-store"

// NoStore marks the response as not cacheable. Member-facing handlers call it first, so error answers carry it too.
func NoStore(c fiber.Ctx) { c.Set(fiber.HeaderCacheControl, CacheControlNoStore) }
