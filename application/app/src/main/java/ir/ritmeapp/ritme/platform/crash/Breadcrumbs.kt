package ir.ritmeapp.ritme.platform.crash

/**
 * In-memory ring buffer of the last [MAX] breadcrumbs (CLAUDE.md §7.5). Custom-written
 * (no library), bounded, and synchronized because it is written from app threads and read
 * from the crashing thread. Ephemeral: never persisted on its own — only embedded into a
 * crash/error report when one is written.
 */
object Breadcrumbs {

    private const val MAX = 20
    private val buffer = ArrayDeque<Breadcrumb>(MAX)

    @Synchronized
    fun add(message: String) {
        if (buffer.size == MAX) buffer.removeFirst()
        buffer.addLast(Breadcrumb(System.currentTimeMillis(), message))
    }

    /** A point-in-time copy, oldest first. Safe to call from the uncaught-exception handler. */
    @Synchronized
    fun snapshot(): List<Breadcrumb> = buffer.toList()

    @Synchronized
    fun clear() = buffer.clear()
}
