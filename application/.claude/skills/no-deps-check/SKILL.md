---
name: no-deps-check
description: Enforce the zero-third-party-library policy of this project. Use before adding any dependency, editing build.gradle, or when tempted to reach for OkHttp/Retrofit/Gson/Room/Hilt/Coil/Lottie/Timber — these are banned and must be hand-written.
---

# No Third-Party Dependencies

CLAUDE.md §3: **default rule is zero third-party libraries.** Only allowed:
- Kotlin standard library
- Android SDK (`android.*`, `androidx.compose.*` for UI primitives only)
- `kotlinx.coroutines` (the single pre-approved exception)
- `org.json` is allowed for JSON because it ships in the Android SDK.

## Banned → write it yourself

| Need | Banned | Hand-write instead |
|---|---|---|
| HTTP | OkHttp, Retrofit | client over `java.net.HttpURLConnection` (+ own request builder, pooling, timeouts) |
| JSON | Gson, Moshi, kotlinx.serialization | hand tokenizer/parser, or `org.json` |
| DB/cache | Room | `SQLiteOpenHelper` + hand DAOs |
| DI | Hilt, Koin, Dagger | manual constructor injection in `adapter/outbound/di` |
| Images | Coil, Glide | `LruCache<String,Bitmap>` + disk cache + `BitmapFactory` on IO |
| Animations | Lottie | Compose `animate*AsState`, `Animatable`, vector drawables |
| Logging | Timber | ~20-line wrapper over `android.util.Log`, build-type gated |
| Collections | any 3rd-party | Kotlin stdlib `List`/`ArrayDeque`; custom only with measured reason |

## Checklist

- [ ] No new line in `build.gradle(.kts)` `dependencies {}` except the allowed set.
- [ ] No banned package imported anywhere.
- [ ] If a library genuinely seems required (e.g. a crypto primitive that must not
      be hand-rolled) → **STOP and ask the user explicitly**. Never add silently.

## How to scan
Read `app/build.gradle*` and check the `dependencies` block. Grep source for
`okhttp|retrofit|gson|moshi|kotlinx.serialization|room|hilt|dagger|koin|coil|glide|lottie|timber`.
Any hit → flag and propose the hand-written replacement above.
